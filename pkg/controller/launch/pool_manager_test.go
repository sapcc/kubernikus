package launch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
)

func TestNodesSorting(t *testing.T) {

	kluster := &v1.Kluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "default",
			Name:      "test-cluster",
		},
	}

	type node struct {
		ID            string
		NodeMissing   bool
		Unschedulable bool
	}

	cases := []struct {
		Nodes  []node
		Result []string
	}{
		{
			Nodes: []node{
				{ID: "id1"},
				{ID: "id2", Unschedulable: true},
				{ID: "id3", NodeMissing: true},
			},
			Result: []string{"id3", "id2", "id1"},
		},
		{
			Nodes: []node{
				{ID: "id1"},
				{ID: "id2", Unschedulable: true},
				{ID: "id3", NodeMissing: true},
				{ID: "id4", NodeMissing: true},
			},
			Result: []string{"id3", "id4", "id2", "id1"},
		},
		{
			Nodes: []node{
				{ID: "id1"},
				{ID: "id2", Unschedulable: true},
				{ID: "id3"},
				{ID: "id4", Unschedulable: true},
				{ID: "id5", Unschedulable: true},
				{ID: "id6", NodeMissing: true},
				{ID: "id7", NodeMissing: true},
			},
			Result: []string{"id6", "id7", "id2", "id4", "id5", "id1", "id3"},
		},
	}

	for _, c := range cases {
		var openStackIDS []string
		var nodes []runtime.Object

		for _, n := range c.Nodes {
			openStackIDS = append(openStackIDS, n.ID)
			if !n.NodeMissing {
				nodes = append(nodes, &core_v1.Node{
					ObjectMeta: meta_v1.ObjectMeta{
						Name: n.ID,
					},
					Spec: core_v1.NodeSpec{
						ProviderID:    "openstack:///" + n.ID,
						Unschedulable: n.Unschedulable,
					},
				})
			}

		}

		pm := ConcretePoolManager{
			nodeObservatory: nodeobservatory.NewFakeController(kluster, nodes...),
			Kluster:         kluster,
		}
		pm.sortByUnschedulableNodes(openStackIDS)
		assert.Equal(t, c.Result, openStackIDS)

	}

}
