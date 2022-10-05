package netutil

import (
	"os/exec"
	"strings"
)

// DefaultInterfaceName returns the name of the interface used by the default route
func DefaultInterfaceName() (string, error) {
	routeCmd := exec.Command("/sbin/route", "-n", "get", "0.0.0.0")
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	// Darwin route out format is always like this:
	//    route to: default
	// destination: default
	//        mask: default
	//     gateway: 192.168.1.1
	//   interface: en5
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "interface:" {
			return fields[1], nil
		}
	}

	return "", NotFound
}
