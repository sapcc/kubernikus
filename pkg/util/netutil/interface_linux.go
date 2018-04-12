package netutil

import (
	"os/exec"
	"strings"
)

//DefaultInterfaceName returns the name of the interface used by the default route
func DefaultInterfaceName() (string, error) {
	ip, err := discoverDefaultInterfaceUsingRoute()
	if err != nil {
		return discoverDefaultInterfaceUsingIp()
	}
	return ip, err
}

func discoverDefaultInterfaceUsingIp() (string, error) {
	output, err := exec.Command("ip", "route", "show").CombinedOutput()
	if err != nil {
		return "", err
	}
	// Linux '/usr/bin/ip route show' format looks like this:
	// default via 192.168.178.1 dev wlp3s0  metric 303
	// 192.168.178.0/24 dev wlp3s0  proto kernel  scope link  src 192.168.178.76  metric 303
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] == "default" {
			if fields[4] != "" {
				return fields[4], nil
			}
		}
	}
	return "", NotFound
}

func discoverDefaultInterfaceUsingRoute() (string, error) {
	output, err := exec.Command("route", "-n").CombinedOutput()
	if err != nil {
		return "", err
	}
	// Linux route out format is always like this:
	// Kernel IP routing table
	// Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
	// 0.0.0.0         192.168.1.1     0.0.0.0         UG    0      0        0 eth0
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 8 && fields[0] == "0.0.0.0" {
			if fields[7] != "" {
				return fields[7], nil
			}
		}
	}
	return "", NotFound
}
