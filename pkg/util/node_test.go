package util

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"

	"github.com/sapcc/kubernikus/pkg/util/netutil"
)

func TestThisNode(t *testing.T) {
	machineIDPath = "machine-id"

	require.NoError(t, ioutil.WriteFile("machine-id", []byte("my-id"), 0644))
	defer func() {
		os.Remove("machine-id")
	}()
	myIP, err := netutil.PrimaryIP()
	require.NoError(t, err)
	myHostname, err := os.Hostname()
	require.NoError(t, err)

	nodeWithMatchingMachineID := v1.Node{Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{MachineID: "my-id"}, Addresses: []v1.NodeAddress{{Type: v1.NodeInternalIP, Address: "1.2.3.4"}}}}
	nodeWithMatchingIP := v1.Node{Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{MachineID: "something"}, Addresses: []v1.NodeAddress{{Type: v1.NodeInternalIP, Address: myIP.String()}}}}
	nodeWithMatchingHostname := v1.Node{Status: v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{MachineID: "something"}, Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: myHostname}}}}

	testCases := []struct {
		test   string
		input  []v1.Node
		output *v1.Node
		err    error
	}{
		{
			"Matches by machine-id",
			[]v1.Node{nodeWithMatchingMachineID},
			&nodeWithMatchingMachineID,
			nil,
		},
		{
			"Matches by IP",
			[]v1.Node{nodeWithMatchingIP},
			&nodeWithMatchingIP,
			nil,
		},
		{
			"Matches by Hostname",
			[]v1.Node{nodeWithMatchingHostname},
			&nodeWithMatchingHostname,
			nil,
		},
		{
			"machine-id has highest priority",
			[]v1.Node{nodeWithMatchingHostname, nodeWithMatchingMachineID, nodeWithMatchingIP},
			&nodeWithMatchingMachineID,
			nil,
		},
		{
			"IP has higher priority then hostname",
			[]v1.Node{nodeWithMatchingHostname, nodeWithMatchingIP},
			&nodeWithMatchingIP,
			nil,
		},
		{
			"Hostname has to be unique",
			[]v1.Node{nodeWithMatchingHostname, nodeWithMatchingHostname},
			nil,
			nil,
		},
	}
	for n, c := range testCases {
		t.Run(c.test, func(t *testing.T) {
			node, _ := ThisNode(c.input)
			assert.Equal(t, c.output, node, "Test number %d (%s) failed", n+1, c.test)
		})
	}
}
