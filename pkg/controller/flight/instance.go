package flight

import (
	"time"

	openstack_kluster "github.com/sapcc/kubernikus/pkg/client/openstack/kluster"
)

type Instance interface {
	GetID() string
	GetName() string
	GetSecurityGroupNames() []string
	GetCreated() time.Time
	Erroring() bool
	Running() bool
	GetPoolName() string
}

type instance struct {
	openstack_kluster.Node
	pool string
}

func (i instance) GetPoolName() string {
	return i.pool
}
