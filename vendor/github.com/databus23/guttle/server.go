package guttle

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/hashicorp/yamux"
	"github.com/oklog/run"
)

//ServerOptions hold the configration of a tunnel server
type ServerOptions struct {
	//Listener is used for accepting tunnel clients.
	//Default: net.Listen("tcp", ":9090")
	Listener net.Listener

	//HijackAddr specifies a local address the server should
	//accept connections on and forward them to tunnel clients.
	//Default: 127.0.0.1:9191
	HijackAddr string

	//ProxyFunc provides a hook for forwarding a tunneled connection to its destination.
	//If non is given NoProxy() is used by default.
	ProxyFunc ProxyFunc

	//Logger used by this service instance
	Logger log.Logger
}

// Server is a tunel Server accepting connections from tunnel clients
type Server struct {
	options *ServerOptions

	clientWg  sync.WaitGroup
	requestWg sync.WaitGroup

	sessions     *sync.Map
	clientRoutes *sync.Map
	randomRoutes *sync.Map

	stop   func()
	logger log.Logger
}

type route struct {
	cidr *net.IPNet
	dest string
}

//NewServer creates a new tunnel server
func NewServer(opts *ServerOptions) *Server {
	if opts == nil {
		opts = new(ServerOptions)
	}
	if opts.HijackAddr == "" {
		opts.HijackAddr = "127.0.0.1:9191"
	}
	if opts.ProxyFunc == nil {
		opts.ProxyFunc = NoProxy()
	}
	logger := opts.Logger
	if logger == nil {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	}

	//TODO: Validate ServerOptions
	return &Server{
		options:      opts,
		sessions:     new(sync.Map),
		clientRoutes: new(sync.Map),
		randomRoutes: new(sync.Map),
		logger:       logger,
	}
}

//Start starts the tunnel server and blocks until it is stopped
func (s *Server) Start() error {

	if s.options.Listener == nil {
		// Accept a TCP connection
		s.logger.Log("msg", "listening for tunnel connections", "addr", ":9090")
		var err error
		s.options.Listener, err = net.Listen("tcp", ":9090")
		if err != nil {
			return err
		}
	}

	s.logger.Log("msg", "listening for redirected connections", "addr", s.options.HijackAddr)
	hijackListener, err := net.Listen("tcp", s.options.HijackAddr)
	if err != nil {
		return fmt.Errorf("Failed to listen on hijack port: %s", err)
	}

	var g run.Group

	//tunnel listener
	g.Add(
		func() error {
			s.logger.Log("msg", "accepting tunnel clients", "addr", s.options.Listener.Addr())
			err := serveListener(s.options.Listener, s.handleClient)
			s.logger.Log("msg", "tunnel listener stopped", "err", err)
			return err
		},
		func(error) {
			s.options.Listener.Close()
		})

	//tcp listener for incoming connections
	g.Add(
		func() error {
			err := serveListener(hijackListener, s.handleHijackedConnection)
			s.logger.Log("msg", "hijack listener stopped", "err", err)
			return err
		},
		func(error) {
			hijackListener.Close()
		})

	// setup close signal
	closeCh := make(chan struct{})
	var once sync.Once
	s.stop = func() { once.Do(func() { close(closeCh) }) }
	g.Add(func() error {
		<-closeCh
		s.logger.Log("msg", "stop signal received")
		return nil
	}, func(error) {
		s.stop()
	})

	return g.Run()
}

//Close signals a running server to stop. It returns immediately.
func (s *Server) Close() {
	s.stop()
}

//AddClientRoute adds a route for specific tunnel client.
//Local tunnel requests for ips that match the given CIDR are routed
//to the specified client.
//If a request is matched by multiple routes a random route is choosen.
func (s *Server) AddClientRoute(cidr string, identifier string) error {
	if identifier == "" {
		return errors.New("Can't add client route for empty identifier")
	}
	_, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.clientRoutes.Store(net.String()+identifier, &route{cidr: net, dest: identifier})
	return nil
}

//DeleteClientRoute deletes a route created with AddClientRoute
func (s *Server) DeleteClientRoute(cidr, identifier string) error {
	if identifier == "" {
		return errors.New("Can't delete client route for empty identifier")
	}
	_, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.clientRoutes.Delete(net.String() + identifier)
	return nil
}

//AddRoute adds a generic route for all client.
//Incoming request that are matched by the given CIDR are routed to
//any active client. Generic routes are considered when no client route
//matched for the given request.
func (s *Server) AddRoute(cidr string) error {
	_, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.randomRoutes.Store(net.String(), net)
	return nil
}

//DeleteRoute deletes a generic route.
func (s *Server) DeleteRoute(cidr string) error {
	_, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	s.randomRoutes.Delete(net.String())
	return nil
}

func (s *Server) getIdentifier(conn net.Conn) (string, error) {

	switch v := conn.(type) {
	case *tls.Conn:
		if peerCerts := v.ConnectionState().PeerCertificates; len(peerCerts) > 0 {
			if cn := peerCerts[0].Subject.CommonName; cn != "" {
				return cn, nil
			}
		}
	}

	return conn.RemoteAddr().String(), nil
}

func (s *Server) handleClient(conn net.Conn) {
	logger := log.With(s.logger, "peer_ip", conn.RemoteAddr())

	defer func() {
		logger.Log("msg", "closing tunnel connection")
		conn.Close()
	}()
	// Ensure the handshake and client verification is done.
	// If we don't call this explicitly its done lazy on first read
	// which is after a tunnel stream has been created.
	if t, ok := conn.(*tls.Conn); ok {
		if err := t.Handshake(); err != nil {
			logger.Log("msg", "tls handshake failed", "err", err)
			return
		}
	}
	identifier, err := s.getIdentifier(conn)
	if err != nil {
		logger.Log("msg", "failed to get identifier", "err", err)
		return
	}
	logger = log.With(logger, "peer", identifier)
	if _, found := s.sessions.Load(identifier); found {
		logger.Log("Already have a session")
		return
	}
	config := yamux.DefaultConfig()
	config.LogOutput = &StdlibAdapter{s.logger}
	session, err := yamux.Server(conn, config)
	if err != nil {
		logger.Log("msg", "failed to create session", "err", err)
	}

	logger.Log("msg", "new tunnel client")

	s.sessions.Store(identifier, session)
	s.clientWg.Add(1)
	defer func() {
		s.sessions.Delete(identifier)
		s.clientWg.Done()
	}()
	for {
		stream, err := session.AcceptStream()
		if err == yamux.ErrSessionShutdown {
			logger.Log("msg", "session shutdown")
			return
		}
		if err != nil {
			logger.Log("msg", "failed accept new stream", "err", err)
			return
		}
		go s.handleIncomingStream(stream, log.With(logger, "src", "tunnel", "stream_id", stream.StreamID()))
	}
}

func (s *Server) handleHijackedConnection(conn net.Conn) {
	logger := log.With(s.logger, "src", conn.RemoteAddr())
	s.requestWg.Add(1)
	defer func() {
		//Might already be closed by Join, ignore error
		conn.Close()
		s.requestWg.Done()
	}()
	originalIP, orginalPort, err := originalDestination(&conn)
	if err != nil {
		logger.Log("msg", "failed to get original destination", "err", err)
		return
	}
	logger = log.With(logger, "destination", fmt.Sprintf("%s:%d", originalIP, orginalPort))

	logger.Log("msg", "accepted connection")

	session := s.route(originalIP, logger)

	if session == nil {
		logger.Log("msg", "rejecting connection", "err", "no active tunnel")
		return
	}
	stream, err := session.OpenStream()
	if err != nil {
		logger.Log("msg", "failed to open stream", err, "err")
	}
	logger = log.With(logger, "stream_id", stream.StreamID())

	err = writeHeader(stream, originalIP, orginalPort)
	if err != nil {
		logger.Log("msg", "Failed to send preamble", "err", err)
	}

	Join(stream, conn, logger)

}

func (s *Server) route(ip net.IP, logger log.Logger) *yamux.Session {
	if session := s.checkClientRoutes(ip, logger); session != nil {
		return session
	}
	//No specific client routes found check global routes
	if session := s.checkRandomRoutes(ip, logger); session != nil {
		return session
	}

	return nil
}

func (s *Server) checkClientRoutes(ip net.IP, logger log.Logger) (session *yamux.Session) {

	s.clientRoutes.Range(func(_, value interface{}) bool {
		routeEntry := value.(*route)
		if routeEntry.cidr.Contains(ip) {
			if s, ok := s.sessions.Load(routeEntry.dest); ok {
				logger.Log("msg", "using client route", "peer", routeEntry.dest)
				session = s.(*yamux.Session)
				return false
			}
			logger.Log("msg", "skipping matching route", "route", routeEntry.cidr, "peer", routeEntry.dest, "err", "no active session")
		}
		return true
	})

	return
}

func (s *Server) checkRandomRoutes(ip net.IP, logger log.Logger) (session *yamux.Session) {
	s.randomRoutes.Range(func(_, val interface{}) bool {
		cidr := val.(*net.IPNet)
		if cidr.Contains(ip) {
			s.sessions.Range(func(key, val interface{}) bool {
				logger.Log("msg", "using random route", "route", cidr, "peer", key)
				session = val.(*yamux.Session)
				return false
			})
		}
		return true
	})

	return

}

func (s *Server) handleIncomingStream(stream *yamux.Stream, logger log.Logger) {
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Log("msg", "failed to close stream", "err", err)
		}
	}()
	if s.options.ProxyFunc == nil {
		logger.Log("msg", "rejecting stream", "err", "no proxy strategy given")
		return
	}
	header, err := readHeader(stream)
	if err != nil {
		logger.Log("msg", "failed to read header from stream", "err", err)
		return
	}
	s.options.ProxyFunc(stream, header, logger)
}
