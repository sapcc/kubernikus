package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/util/netutil"
)

const (
	NODE_COREOS_PREFIX     = "Container Linux by CoreOS"
	NODE_FLATCAR_PREFIX    = "Flatcar Container Linux"
	NODEPOOL_COREOS_IMAGE  = "coreos-stable-amd64"
	NODEPOOL_FLATCAR_IMAGE = "flatcar-stable-amd64"
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

type annotations struct {
	Metadata struct {
		Annotations map[string]interface{} `json:"annotations"`
	} `json:"metadata"`
}

func AddNodeAnnotation(nodeName, key, val string, client kubernetes.Interface) error {
	var a annotations
	a.Metadata.Annotations = map[string]interface{}{key: val}
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("Failed to marshal annotation %v = %v: %s", key, val, err)
	}
	_, err = client.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, data)
	return err
}

func RemoveNodeAnnotation(nodeName, key string, client kubernetes.Interface) error {
	var a annotations
	a.Metadata.Annotations = map[string]interface{}{key: nil}
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("Failed to marshal annotation %v = %v: %s", key, nil, err)
	}
	_, err = client.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, data)
	return err
}

func IsCoreOSNode(node *v1.Node) bool {
	return strings.HasPrefix(node.Status.NodeInfo.OSImage, NODE_COREOS_PREFIX)
}

func IsFlatcarNode(node *v1.Node) bool {
	return strings.HasPrefix(node.Status.NodeInfo.OSImage, NODE_FLATCAR_PREFIX)
}

func IsCoreOSNodePool(pool *models.NodePool) bool {
	return pool.Image == NODEPOOL_COREOS_IMAGE
}

func IsFlatcarNodePool(pool *models.NodePool) bool {
	return pool.Image == NODEPOOL_FLATCAR_IMAGE
}
