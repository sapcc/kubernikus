package guttle

import (
	"net"
	"time"

	"github.com/go-kit/kit/log"
)

// ProxyFunc is responsible for forwarding a tunneled connection to a local destination and writing the response back.
type ProxyFunc func(remote net.Conn, hdr Header, logger log.Logger)

// NoProxy returns a ProxyFunc that does nothing
func NoProxy() ProxyFunc {
	return func(src net.Conn, _ Header, logger log.Logger) {
		logger.Log("msg", "rejecting connection")
		src.Close()
	}
}

// SourceRoutedProxy returns a ProxyFunc that honors the header information
// of the proxied request and forwards traffic to the given header information.
func SourceRoutedProxy() ProxyFunc {
	return func(src net.Conn, hdr Header, logger log.Logger) {
		destination := hdr.Destination()
		conn, err := net.DialTimeout("tcp", dest, 5*time.Second)
		if err != nil {
			logger.Log("msg", "connection failed", "dest", destination, "err", err)
			return
		}
		logger = log.With(logger, "src", conn.LocalAddr(), "dest", dest)
		//Note: Join closes the connection
		Join(src, conn, logger)
	}
}

// StaticProxy ignores the request header and forwards traffic to a static destination
func StaticProxy(destination string) ProxyFunc {
	return func(src net.Conn, _ Header, logger log.Logger) {
		conn, err := net.DialTimeout("tcp", destination, 5*time.Second)
		if err != nil {
			logger.Log("msg", "connection failed", "dest", destination, "err", err)
			return
		}
		logger = log.With(logger, "src", conn.LocalAddr(), "dest", destination)
		//Note: Join closes the connection
		Join(src, conn, logger)
	}
}
