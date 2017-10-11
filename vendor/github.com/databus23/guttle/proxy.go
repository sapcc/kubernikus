package guttle

import (
	"log"
	"net"
)

// ProxyFunc is responsible for forwarding a tunneled connection to a local destination and writing the response back.
type ProxyFunc func(remote net.Conn, hdr Header)

// NoProxy returns a ProxyFunc that does nothing
func NoProxy() ProxyFunc {
	return func(src net.Conn, _ Header) {
		log.Print("Rejecting connection")
		src.Close()
	}
}

// SourceRoutedProxy returns a ProxyFunc that honors the header information
// of the proxied request and forwards traffic to the given header information.
func SourceRoutedProxy() ProxyFunc {
	return func(src net.Conn, hdr Header) {
		dest := net.TCPAddr{IP: hdr.DestinationIP(), Port: hdr.DestinationPort()}
		conn, err := net.DialTCP("tcp", nil, &dest)
		if err != nil {
			log.Printf("Failed to connect to %s: %s", dest.String(), err)
			return
		}
		defer func() {
			if err := conn.Close(); err != nil {
				log.Printf("Error closing connection: %s", err)
			}
		}()
		Join(src, conn)
	}
}

// StaticProxy ignores the request header and forwards traffic to a static destination
func StaticProxy(destination string) ProxyFunc {
	return func(src net.Conn, _ Header) {
		conn, err := net.Dial("tcp", destination)
		if err != nil {
			log.Printf("Failed to connect to %s: %s", destination, err)
			return
		}
		defer func() {
			if err := conn.Close(); err != nil {
				log.Printf("Error closing connection: %s", err)
			}
		}()
		Join(src, conn)
	}
}
