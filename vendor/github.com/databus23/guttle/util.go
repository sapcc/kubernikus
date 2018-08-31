package guttle

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
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
func Join(local, remote net.Conn, logger log.Logger) {
	var wg sync.WaitGroup
	wg.Add(2)

	var sendBytes, receivedBytes int64

	transfer := func(side string, dst, src net.Conn, transferedBytes *int64) {
		var err error

		*transferedBytes, err = io.Copy(dst, src)
		if err != nil {
			logger.Log("msg", "copy error", "side", side, "err", err)
		}
		if d, ok := dst.(*net.TCPConn); ok {
			if err := d.CloseWrite(); err != nil {
				logger.Log("msg", "failed to closeWrite dst", "side", side, "err", err)
			}
		} else {
			if err := dst.Close(); err != nil {
				logger.Log("msg", "failed to dst", "side", side, "err", err)
			}
		}
		//logger.Log("msg", "done", "side", side)

		wg.Done()
	}

	logger.Log("msg", "start proxying")
	start := time.Now()
	go transfer("remote to local", local, remote, &receivedBytes)
	go transfer("local to remote", remote, local, &sendBytes)
	wg.Wait()

	if r, ok := remote.(*net.TCPConn); ok {
		if err := r.Close(); err != nil {
			logger.Log("msg", "closing tcp connection failed", "err", err)
		}
	}
	elapsed := time.Since(start)
	logger.Log("msg", "end proxying", "bytes_send", sendBytes, "bytes_received", receivedBytes, "duration", elapsed)
}
