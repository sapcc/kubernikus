package guttle

import (
	"errors"
	"log"
	"net"
)

func originalDestination(conn *net.Conn) (ip net.IP, port uint16, err error) {
	log.Print("original Destination not not implemented on this platform, returning local addr")

	tcpAddr, ok := ((*conn).LocalAddr()).(*net.TCPAddr)
	if !ok {
		return ip, port, errors.New("not a TCPConn")
	}

	return tcpAddr.IP, uint16(tcpAddr.Port), nil

}
