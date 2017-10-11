package guttle

import (
	"errors"
	"net"
	"syscall"
	"unsafe"
)

type sockaddr struct {
	family uint16
	data   [14]byte
}

const SO_ORIGINAL_DST = 80

// originalDestination returns an intercepted connection's original destination.
func originalDestination(conn *net.Conn) (ip net.IP, port uint16, err error) {
	tcpConn, ok := (*conn).(*net.TCPConn)
	if !ok {
		return ip, port, errors.New("not a TCPConn")
	}

	file, err := tcpConn.File()
	if err != nil {
		return
	}

	// To avoid potential problems from making the socket non-blocking.
	tcpConn.Close()
	*conn, err = net.FileConn(file)
	if err != nil {
		return
	}

	defer file.Close()
	fd := file.Fd()

	var addr sockaddr
	size := uint32(unsafe.Sizeof(addr))
	err = getsockopt(int(fd), syscall.SOL_IP, SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&addr)), &size)
	if err != nil {
		return
	}

	switch addr.family {
	case syscall.AF_INET:
		ip = addr.data[2:6]
	default:
		return ip, port, errors.New("unrecognized address family")
	}

	port = uint16(addr.data[0])<<8 + uint16(addr.data[1])

	return
}

func getsockopt(s int, level int, name int, val uintptr, vallen *uint32) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(name), uintptr(val), uintptr(unsafe.Pointer(vallen)), 0)
	if e1 != 0 {
		err = e1
	}
	return
}
