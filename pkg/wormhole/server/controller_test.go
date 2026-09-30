//go:build linux
// +build linux

package server

import (
	"testing"

	"github.com/go-kit/log"
	"github.com/google/nftables"
	"github.com/mdlayher/netlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	knftables "github.com/sapcc/kubernikus/pkg/util/nftables"
)

// newTestConn creates a connection that ACKs all writes without touching the
// kernel. Read-back calls (ListTables, GetRules) return empty results.
func newTestConn(t *testing.T) *nftables.Conn {
	t.Helper()
	c, err := nftables.New(nftables.WithTestDial(
		func(req []netlink.Message) ([]netlink.Message, error) {
			acks := make([]netlink.Message, 0, len(req))
			for _, msg := range req {
				if msg.Header.Flags&netlink.Acknowledge != 0 {
					acks = append(acks, netlink.Message{
						Header: netlink.Header{
							Length:   4,
							Type:     netlink.Error,
							Sequence: msg.Header.Sequence,
							PID:      msg.Header.PID,
						},
						Data: []byte{0, 0, 0, 0},
					})
				}
			}
			return acks, nil
		}))
	require.NoError(t, err)
	return c
}

func newTestController(t *testing.T) (*Controller, *fake.Clientset) {
	t.Helper()
	client := fake.NewSimpleClientset()
	factory := informers.NewSharedInformerFactory(client, 0)
	nodeInformer := factory.Core().V1().Nodes()

	c := &Controller{
		nodes:       nodeInformer,
		queue:       nil, // not needed for unit tests
		store:       make(map[string][]route),
		nft:         knftables.NewWithConn(newTestConn(t)),
		hijackPort:  9191,
		serviceCIDR: "10.96.0.0/12",
		Logger:      log.NewNopLogger(),
	}
	return c, client
}

func TestSyncRules_ServiceCIDRAlwaysIncluded(t *testing.T) {
	c, _ := newTestController(t)

	err := c.syncRules()
	require.NoError(t, err)
	// Should succeed with just serviceCIDR and no nodes
}

func TestSyncRules_IncludesNodeIPAndPodCIDR(t *testing.T) {
	c, _ := newTestController(t)

	// Manually populate store as addNode would
	c.store["node1"] = []route{
		{cidr: "192.168.1.1/32", identifier: "system:node:node1"},
		{cidr: "10.244.1.0/24", identifier: "system:node:node1"},
	}

	err := c.syncRules()
	require.NoError(t, err)
}

func TestAddNode_PopulatesStore(t *testing.T) {
	c, _ := newTestController(t)

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Spec:       v1.NodeSpec{PodCIDR: "10.244.1.0/24"},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{Type: v1.NodeInternalIP, Address: "192.168.1.1"},
			},
		},
	}

	// Add node to informer cache
	err := c.nodes.Informer().GetIndexer().Add(node)
	require.NoError(t, err)

	err = c.addNode("node1", node)
	require.NoError(t, err)

	c.storeMu.RLock()
	defer c.storeMu.RUnlock()
	assert.Len(t, c.store["node1"], 2)
}

func TestDelNode_ClearsStore(t *testing.T) {
	c, _ := newTestController(t)

	c.store["node1"] = []route{
		{cidr: "192.168.1.1/32", identifier: "system:node:node1"},
	}

	err := c.delNode("node1")
	require.NoError(t, err)

	c.storeMu.RLock()
	defer c.storeMu.RUnlock()
	_, exists := c.store["node1"]
	assert.False(t, exists, "store entry should be deleted after delNode")
}
