package guttle

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/databus23/guttle/group"
	"github.com/hashicorp/yamux"
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
}

// Client is responsible for connecting to a tunnel server.
type Client struct {
	options   *ClientOptions
	session   *yamux.Session
	requestWg sync.WaitGroup
	stop      func()
}

// NewClient create a new Client
func NewClient(opts *ClientOptions) *Client {
	if opts.Backoff == nil {
		opts.Backoff = newForeverBackoff()
	}
	if opts.ProxyFunc == nil {
		opts.ProxyFunc = SourceRoutedProxy()
	}
	return &Client{options: opts}
}

// Start the client and connects to the server
func (c *Client) Start() error {

	var g group.Group
	if c.options.ListenAddr != "" {
		listener, err := net.Listen("tcp", c.options.ListenAddr)
		if err != nil {
			return err
		}
		g.Add(func() error {
			err := serveListener(listener, c.handleConnection)
			log.Printf("Stopped listener %s", listener.Addr())
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
		log.Print("Stop channel closed")
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
			log.Print("Shutdown detected")
			return nil
		}
		dur := c.options.Backoff.NextBackOff()
		if dur < 0 {
			return ErrRedialAborted
		}
		log.Printf("Connection failure: %s. Backing off for %s", err, dur)
		select {
		case <-time.NewTimer(dur).C:
		case <-close:
			return nil

		}
	}
}

func (c *Client) handleConnection(conn net.Conn) {
	defer func() {
		log.Printf("Closing connection %s", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			log.Printf("Close error: %s", err)
		}
	}()

	originalIP, orginalPort, err := originalDestination(&conn)
	if err != nil {
		log.Printf("Failed to get original destination address: %s", err)
		return
	}

	log.Printf("Accepted connection %s -> %s:%d", conn.RemoteAddr(), originalIP, orginalPort)

	if c.session == nil {
		log.Print("No active tunnel. Rejecting")
		return
	}
	stream, err := c.session.OpenStream()
	if err != nil {
		log.Printf("Failed to open stream: %s", err)
		return
	}
	err = writeHeader(stream, originalIP, orginalPort)
	if err != nil {
		log.Printf("Failed to send preamble: %s", err)
	}

	Join(stream, conn)
}

func (c *Client) closeTunnel() {
	if c.session == nil {
		return
	}
	// wait until all connections are finished
	waitCh := make(chan struct{})
	go func() {
		if err := c.session.GoAway(); err != nil {
			log.Printf("Session go away failed: %s", err)
		}

		c.requestWg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
		// ok
	case <-time.After(time.Second * 10):
		log.Print("Timeout waiting for connections to finish")
	}

	if err := c.session.Close(); err != nil {
		log.Printf("Error closing session: %s", err)
	}

}

func (c *Client) connect() error {
	log.Printf("Connecting to %s", c.options.ServerAddr)
	conn, err := c.dial(c.options.ServerAddr)
	if err != nil {
		return err
	}

	// Setup client side of yamux
	session, err := yamux.Client(conn, nil)
	if err != nil {
		log.Fatal(err)
	}
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
	log.Printf("Handeling stream %d from %s", stream.StreamID(), stream.RemoteAddr())

	c.requestWg.Add(1)
	defer func() {
		stream.Close() //check: stream might be closed twice
		c.requestWg.Done()
	}()
	header, err := readHeader(stream)
	if err != nil {
		log.Printf("Failed to parse header: %s", err)
		return
	}
	c.options.ProxyFunc(stream, header)
}

func (c *Client) dial(serverAddr string) (net.Conn, error) {
	if c.options.Dial != nil {
		return c.options.Dial("tcp", serverAddr)
	}
	return net.Dial("tcp", serverAddr)
}
