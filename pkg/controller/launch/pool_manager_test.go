package launch

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/stretchr/testify/assert"
)

type fakePoolNode struct {
	running bool
}

func (f *fakePoolNode) Running() bool {
	return f.running
}

func TestRunning(t *testing.T) {

	var clients config.Clients
	kluster := &v1.Kluster{}
	logger := log.NewNopLogger()
	pool := &models.NodePool{
		Config: models.NodePoolConfig{
			Repair:  false,
			Upgrade: false,
		},
		Flavor: "hase",
		Image:  "kuh",
		Name:   "maus",
		Size:   3,
	}

	pm := &ConcretePoolManager{clients, kluster, pool, logger}

	nodes := []openstack.Node{}
	assert.Equal(t, 0, pm.running(nodes))

	nodes = []PoolNode{
		fakePoolNode{true},
		fakePoolNode{false},
	}

	assert.Equal(t, 1, pm.running(nodes))
}
