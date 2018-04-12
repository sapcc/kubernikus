package util

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/api/core/v1"

	"github.com/sapcc/kubernikus/pkg/util/netutil"
)

//Taken from https://github.com/kubernetes/kubernetes/blob/886e04f1fffbb04faf8a9f9ee141143b2684ae68/pkg/api/v1/node/util.go
// IsNodeReady returns true if a node is ready; false otherwise.
func IsNodeReady(node *v1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == v1.NodeReady {
			return c.Status == v1.ConditionTrue
		}
	}
	return false
}

// ThisNode identifies the node object that is representing the system ThisNode is execited on.
func ThisNode(nodes []v1.Node) (*v1.Node, error) {

	//1. try to match by machineID
	machineID, err := GetMachineID()
	//blacklist machine-ids baked into v1.7.5_coreos.0, v1.8.5_coreos.0 and v1.9.0_coreos.0 hyperkube images
	for _, blacklisted := range []string{"833e0926ee21aed71ec075d726cbcfe0", "40beb5eb909e171860ceee669da56e1d"} {
		if machineID == blacklisted {
			err = errors.New("machine-id blacklisted")
			break
		}
	}
	if err == nil {
		for _, node := range nodes {
			if node.Status.NodeInfo.MachineID == machineID {
				return &node, nil
			}
		}
	}

	//2. try to match by ip
	if internalIP, err := netutil.PrimaryIP(); err == nil {
		for _, node := range nodes {
			for _, address := range node.Status.Addresses {
				if address.Type == v1.NodeInternalIP && address.Address == internalIP.String() {
					return &node, nil
				}
			}
		}
	}

	//3. try to match by hostname
	var candidate *v1.Node
	if hostname, err := os.Hostname(); err == nil {
	Loop:
		for _, node := range nodes {
			for _, address := range node.Status.Addresses {
				if address.Type == v1.NodeHostName && hostname == address.Address {
					//if we find more than one node with the same hostname, bail
					if candidate != nil {
						candidate = nil
						break Loop
					}
					candidate = &node
					break
				}
			}
		}
		if candidate != nil {
			return candidate, nil
		}
	}

	return nil, errors.New("Not found")
}

var machineIDPath = "/etc/machine-id"

// GetMachineID returns a host's 128-bit machine ID as a string. This functions
// similarly to systemd's `sd_id128_get_machine`: internally, it simply reads
// the contents of /etc/machine-id
// http://www.freedesktop.org/software/systemd/man/sd_id128_get_machine.html

func GetMachineID() (string, error) {
	machineID, err := ioutil.ReadFile(machineIDPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %v", machineIDPath, err)
	}
	return strings.TrimSpace(string(machineID)), nil
}
