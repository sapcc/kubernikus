package guttle

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/hashicorp/yamux"
	"github.com/oklog/run"
)

//ErrRedialAborted is returned by Client.Start when the configured
//backoff strategy gives up.
var ErrRedialAborted = errors.New("unable to restore the connection, aborting")

// ClientOptions defines the configuration for the Client
type ClientOptions struct {
	// ServerAddr defines the TCP address of the tunnel server to be connected.
	ServerAddr string
	// Dial provides custom transport layer for client server communication.
	Dial func(network, address string) (net.Conn, error)

	//ProxyFunc provides a hook for forwarding a tunneled connection to its destination.
	//If non is given SourceRoutedProxy() is used by default.
	ProxyFunc ProxyFunc

	//ListenAddr optinally lets the Client open a listening socket and forward traffic
	//received on the listener to the tunnel server.
	ListenAddr string

	// Backoff is used to control behavior of staggering reconnection loop.
	//
	// If nil, default backoff policy is used which makes a client to never
	// give up on reconnection.
	//
	// If custom backoff is used, Start() will return ErrRedialAborted set
	// with ClientClosed event when no more reconnection atttemps should
	// be made.
	Backoff Backoff

	//Logger used by this service instance
	Logger log.Logger
}

// Client is responsible for connecting to a tunnel server.
type Client struct {
	options   *ClientOptions
	session   *yamux.Session
	requestWg sync.WaitGroup
	stop      func()
	logger    log.Logger
}

// NewClient create a new Client
func NewClient(opts *ClientOptions) *Client {
	if opts.Backoff == nil {
		opts.Backoff = newForeverBackoff()
	}
	if opts.ProxyFunc == nil {
		opts.ProxyFunc = SourceRoutedProxy()
	}
	logger := opts.Logger
	if logger == nil {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	}

	return &Client{options: opts, logger: logger}
}

// Start the client and connects to the server
func (c *Client) Start() error {

	var g run.Group
	if c.options.ListenAddr != "" {
		listener, err := net.Listen("tcp", c.options.ListenAddr)
		if err != nil {
			return err
		}
		g.Add(func() error {
			c.logger.Log("msg", "listening for connections", "addr", listener.Addr())
			err := serveListener(listener, c.handleConnection)
			c.logger.Log("msg", "stopped", "listener", listener.Addr())
			return err
		}, func(error) {
			listener.Close()
		})
	}

	stopTunnel := make(chan struct{})

	g.Add(func() error {
		return c.serveTunnel(stopTunnel)
	},
		func(error) {
			close(stopTunnel)
		})

	stopCh := make(chan struct{})
	var once sync.Once
	c.stop = func() {
		once.Do(func() { close(stopCh) })
	}
	g.Add(func() error {
		<-stopCh
		return nil
	}, func(error) {
		c.stop()
	})
	return g.Run()

}

//Stop signals a running client to disconnect. It returns immediately.
func (c *Client) Stop() {
	c.stop()
}

func (c *Client) serveTunnel(close <-chan struct{}) error {

	go func() {
		<-close
		c.closeTunnel()
	}()
	for {
		err := c.connect()
		if err == nil {
			c.logger.Log("msg", "shutdown")
			return nil
		}
		dur := c.options.Backoff.NextBackOff()
		if dur < 0 {
			return ErrRedialAborted
		}
		c.logger.Log("msg", "connection failure", "backoff", dur, "err", err)
		select {
		case <-time.NewTimer(dur).C:
		case <-close:
			return nil

		}
	}
}

func (c *Client) handleConnection(conn net.Conn) {

	logger := log.With(c.logger, "src", conn.RemoteAddr())

	defer func() {
		//don't check the error, connection might alrady by closed
		conn.Close()
	}()

	originalIP, orginalPort, err := originalDestination(&conn)
	if err != nil {
		c.logger.Log("msg", "failed to get original destination", "err", err)
		return
	}
	logger = log.With(logger, "destination", fmt.Sprintf("%s:%d", originalIP, orginalPort))

	if c.session == nil {
		logger.Log("err", "No active tunnel. Rejecting")
		return
	}
	stream, err := c.session.OpenStream()
	if err != nil {
		logger.Log("msg", "failed to open stream", "err", err)
		return
	}
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Log("msg", "failed to close stream", "err", err)
		}
	}()
	logger = log.With(logger, "stream_id", stream.StreamID())
	err = writeHeader(stream, originalIP, orginalPort)
	if err != nil {
		logger.Log("msg", "Failed to send preamble", "err", err)
	}

	Join(stream, conn, logger)
}

func (c *Client) closeTunnel() {
	if c.session == nil {
		return
	}
	// wait until all connections are finished
	waitCh := make(chan struct{})
	go func() {
		if err := c.session.GoAway(); err != nil {
			c.logger.Log("msg", "Session go away failed", "err", err)
		}

		c.requestWg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
		// ok
	case <-time.After(time.Second * 10):
		c.logger.Log("err", "Timeout waiting for connections to finish")
	}

	c.logger.Log("msg", "closing session", "err", c.session.Close())

}

func (c *Client) connect() error {
	c.logger.Log("msg", "connecting", "remote", c.options.ServerAddr)
	conn, err := c.dial(c.options.ServerAddr)
	if err != nil {
		return err
	}
	// Setup client side of yamux
	config := yamux.DefaultConfig()
	config.LogOutput = &StdlibAdapter{c.logger}
	session, err := yamux.Client(conn, config)
	if err != nil {
		c.logger.Log("msg", "failed to create client", "err", err)
		os.Exit(1)
	}
	c.logger.Log("msg", "connected")
	c.session = session
	defer func() { c.session = nil }()

	c.options.Backoff.Reset() // we successfully connected, so we can reset the backoff

	for {
		stream, err := session.AcceptStream()
		if err == yamux.ErrSessionShutdown {
			return nil
		}
		if err != nil {
			return err
		}

		go c.handleStream(stream)
	}
}

func (c *Client) handleStream(stream *yamux.Stream) {

	logger := log.With(c.logger, "src", "tunnel", "stream_id", stream.StreamID())

	c.requestWg.Add(1)
	defer func() {
		stream.Close() //check: stream might be closed twice
		c.requestWg.Done()
	}()
	header, err := readHeader(stream)
	if err != nil {
		logger.Log("msg", "Failed to parse header", "msg", err)
		return
	}
	c.options.ProxyFunc(stream, header, logger)
}

func (c *Client) dial(serverAddr string) (net.Conn, error) {
	if c.options.Dial != nil {
		return c.options.Dial("tcp", serverAddr)
	}
	return net.Dial("tcp", serverAddr)
}
