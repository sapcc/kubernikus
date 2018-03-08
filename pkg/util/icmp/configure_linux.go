package icmp

import (
	"net"
	"strings"
	"syscall"
)

func configureSocket(fd int, address string) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, intf := range interfaces {
		addrs, err := intf.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if strings.HasPrefix(addr.String(), address) {
				return syscall.BindToDevice(fd, intf.Name)
			}
		}
	}
	return nil
}
