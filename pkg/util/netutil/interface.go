package netutil

import (
	"errors"
	"net"
)

var NotFound = errors.New("Not found")

//InterfaceAddress returns the first ipv4 address of the given interface
func InterfaceAddress(name string) (net.IP, error) {
	intf, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := intf.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if v4 := ipnet.IP.To4(); v4 != nil {
				return v4, nil
			}
		}
	}
	return nil, NotFound
}

//PrimaryIP returns the first ipv4 Address of the interface which is used by the default route
func PrimaryIP() (net.IP, error) {
	intf, err := DefaultInterfaceName()
	if err != nil {
		return nil, err
	}
	return InterfaceAddress(intf)
}
