package guttle

import (
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
)

func serveListener(listener net.Listener, handler func(conn net.Conn)) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go handler(conn)
	}
}

// Join copies data between local and remote connections.
// It reads from one connection and writes to the other.
// It's a building block for ProxyFunc implementations.
func Join(local, remote net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	var sendBytes, receivedBytes int64

	transfer := func(side string, dst, src net.Conn, transferedBytes *int64) {
		//log.Printf("proxing %s %s -> %s", side, src.RemoteAddr(), dst.RemoteAddr())
		var err error

		*transferedBytes, err = io.Copy(dst, src)
		if err != nil {
			log.Printf("%s: copy error: %s", side, err)
		}

		if err := src.Close(); err != nil {
			log.Printf("%s: close error: %s", side, err)
		}
		if d, ok := dst.(*yamux.Stream); ok {
			d.Close()
		}

		// not for yamux streams, but for client to local server connections
		if d, ok := dst.(*net.TCPConn); ok {
			if err := d.CloseWrite(); err != nil {
				log.Printf("%s: closeWrite error: %#v", side, err)
			}
		}
		wg.Done()
		//log.Printf("done proxing %s %s -> %s: %d bytes", side, src.RemoteAddr(), dst.RemoteAddr(), n)
	}

	log.Printf("proxing %s <-> %s", local.RemoteAddr(), remote.RemoteAddr())
	start := time.Now()
	go transfer("remote to local", local, remote, &receivedBytes)
	go transfer("local to remote", remote, local, &sendBytes)
	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("done proxing %s <-> %s: send %d bytes, received %d bytes, elapsed %s", local.RemoteAddr(), remote.RemoteAddr(), sendBytes, receivedBytes, elapsed)
}
