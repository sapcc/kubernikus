package guttle

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/databus23/guttle/group"
	"github.com/hashicorp/yamux"
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
}

// Server is a tunel Server accepting connections from tunnel clients
type Server struct {
	options *ServerOptions

	clientWg  sync.WaitGroup
	requestWg sync.WaitGroup

	sessions     *sync.Map
	clientRoutes *sync.Map
	randomRoutes *sync.Map

	stop func()
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

	//TODO: Validate ServerOptions
	return &Server{
		options:      opts,
		sessions:     new(sync.Map),
		clientRoutes: new(sync.Map),
		randomRoutes: new(sync.Map),
	}
}

//Start starts the tunnel server and blocks until it is stopped
func (s *Server) Start() error {

	if s.options.Listener == nil {
		// Accept a TCP connection
		log.Printf("Listening for tunnel connections on %s", ":9090")
		var err error
		s.options.Listener, err = net.Listen("tcp", ":9090")
		if err != nil {
			return err
		}
	}

	log.Printf("Listening for redirected connections on %s", s.options.HijackAddr)
	hijackListener, err := net.Listen("tcp", s.options.HijackAddr)
	if err != nil {
		return fmt.Errorf("Failed to listen on hijack port: %s", err)
	}

	var g group.Group

	//tunnel listener
	g.Add(
		func() error {
			err := serveListener(s.options.Listener, s.handleClient)
			log.Printf("Tunnel listener stopped: %s", err)
			return err
		},
		func(error) {
			s.options.Listener.Close()
		})

	//tcp listener for incoming connections
	g.Add(
		func() error {
			err := serveListener(hijackListener, s.handleHijackedConnection)
			log.Printf("Hijack listener stopped: %s", err)
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
		log.Print("Stop signal received")
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
	defer func() {
		log.Printf("Closing connection for tunnel client %s", conn.RemoteAddr())
		conn.Close()
	}()
	identifier, err := s.getIdentifier(conn)
	if err != nil {
		log.Printf("Failed to get identifier: %s", err)
		return
	}
	if _, found := s.sessions.Load(identifier); found {
		log.Printf("Already have a session for %s", identifier)
		return
	}
	session, err := yamux.Server(conn, nil)
	if err != nil {
		log.Printf("Failed to create session for %s", identifier)
	}
	log.Printf("Accepted tunnel client (id=%s) from %s", identifier, conn.RemoteAddr())
	s.sessions.Store(identifier, session)
	s.clientWg.Add(1)
	defer func() {
		s.sessions.Delete(identifier)
		s.clientWg.Done()
	}()
	for {
		stream, err := session.AcceptStream()
		if err == yamux.ErrSessionShutdown {
			log.Printf("Session shutdown for %s", identifier)
			return
		}
		if err != nil {
			log.Printf("Unknown error on session for %s: %#v", identifier, err)
			return
		}
		go s.handleIncomingStream(stream)
	}
}

func (s *Server) handleHijackedConnection(conn net.Conn) {
	s.requestWg.Add(1)
	defer func() {
		log.Printf("Closing connection %s", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			log.Printf("Close error: %s", err)
		}
		s.requestWg.Done()
	}()
	originalIP, orginalPort, err := originalDestination(&conn)
	if err != nil {
		log.Printf("Failed to get original destination address: %s", err)
		return
	}
	log.Printf("Accepted connection %s -> %s:%d", conn.RemoteAddr(), originalIP, orginalPort)

	session := s.route(originalIP)

	if session == nil {
		log.Printf("No active session. Rejecting")
		return
	}
	stream, err := session.OpenStream()
	if err != nil {
		log.Printf("Failed to open stream: %s", err)
	}
	err = writeHeader(stream, originalIP, orginalPort)
	if err != nil {
		log.Printf("Failed to send preamble: %s", err)
	}

	Join(stream, conn)

}

func (s *Server) route(ip net.IP) *yamux.Session {
	if session := s.checkClientRoutes(ip); session != nil {
		return session
	}
	//No specific client routes found check global routes
	if session := s.checkRandomRoutes(ip); session != nil {
		return session
	}

	return nil
}

func (s *Server) checkClientRoutes(ip net.IP) (session *yamux.Session) {

	s.clientRoutes.Range(func(_, value interface{}) bool {
		routeEntry := value.(*route)
		if routeEntry.cidr.Contains(ip) {
			if s, ok := s.sessions.Load(routeEntry.dest); ok {
				session = s.(*yamux.Session)
				return false
			}
			log.Printf("Skipping matching route %s, no active session found", routeEntry)
		}
		return true
	})

	return
}

func (s *Server) checkRandomRoutes(ip net.IP) (session *yamux.Session) {
	s.randomRoutes.Range(func(_, val interface{}) bool {
		cidr := val.(*net.IPNet)
		if cidr.Contains(ip) {
			s.sessions.Range(func(key, val interface{}) bool {
				session = val.(*yamux.Session)
				return false
			})
		}
		return true
	})

	return

}

func (s *Server) handleIncomingStream(stream *yamux.Stream) {
	log.Printf("Handeling stream %d from %s", stream.StreamID(), stream.RemoteAddr())
	defer stream.Close()
	if s.options.ProxyFunc == nil {
		return
	}
	header, err := readHeader(stream)
	if err != nil {
		log.Printf("Failed to read header: %s", err)
		return
	}
	s.options.ProxyFunc(stream, header)
}
