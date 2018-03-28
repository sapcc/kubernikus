package flight

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	openstack_kluster "github.com/sapcc/kubernikus/pkg/client/openstack/kluster"
)

type fakeInstance struct {
	ID                 string
	Name               string
	Created            time.Time
	SecurityGroupNames []string
}

func (f *fakeInstance) GetID() string {
	return f.ID
}

func (f *fakeInstance) GetName() string {
	return f.Name
}

func (f *fakeInstance) GetSecurityGroupNames() []string {
	return f.SecurityGroupNames
}

func (f *fakeInstance) GetCreated() time.Time {
	return f.Created
}

type MockKlusterClient struct {
	mock.Mock
}

func (m *MockKlusterClient) CreateNode(pool *models.NodePool, nodeName string, data []byte) (string, error) {
	args := m.Called(pool, nodeName, data)
	return args.String(0), args.Error(1)
}

func (m *MockKlusterClient) DeleteNode(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockKlusterClient) ListNodes(pool *models.NodePool) ([]openstack_kluster.Node, error) {
	args := m.Called(pool)
	return args.Get(0).([]openstack_kluster.Node), args.Error(1)
}

func (m *MockKlusterClient) SetSecurityGroup(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockKlusterClient) EnsureKubernikusRuleInSecurityGroup() (created bool, err error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func TestEnsureInstanceSecurityGroupAssignment(t *testing.T) {
	kluster := &v1.Kluster{
		Spec: models.KlusterSpec{
			Openstack: models.OpenstackSpec{
				SecurityGroupName: "custom",
			},
		},
	}

	instances := []Instance{
		&fakeInstance{ID: "a", SecurityGroupNames: []string{"default"}},
		&fakeInstance{ID: "b", SecurityGroupNames: []string{}},
		&fakeInstance{ID: "c", SecurityGroupNames: []string{}},
		&fakeInstance{ID: "d", SecurityGroupNames: []string{"custom"}},
		&fakeInstance{ID: "e", SecurityGroupNames: []string{"default", "custom"}},
	}

	nodes := []*core_v1.Node{}

	client := &MockKlusterClient{}
	client.On("SetSecurityGroup", "a").Return(nil)
	client.On("SetSecurityGroup", "b").Return(fmt.Errorf("Boom"))
	client.On("SetSecurityGroup", "c").Return(nil)

	reconciler := flightReconciler{
		kluster,
		instances,
		nodes,
		client,
		log.NewNopLogger(),
	}

	ids := reconciler.EnsureInstanceSecurityGroupAssignment()
	client.AssertCalled(t, "SetSecurityGroup", "a")
	client.AssertCalled(t, "SetSecurityGroup", "b")
	client.AssertCalled(t, "SetSecurityGroup", "c")
	client.AssertNotCalled(t, "SetSecurityGroup", "d")
	client.AssertNotCalled(t, "SetSecurityGroup", "e")
	assert.ElementsMatch(t, ids, []string{"a", "c"})
}

func TestDeleteIncompletelySpawnedInstances(t *testing.T) {
	kluster := &v1.Kluster{}

	instances := []Instance{
		&fakeInstance{ID: "a", Name: "a", Created: time.Now().Add(-24 * time.Hour)},
		&fakeInstance{ID: "b", Name: "b", Created: time.Now().Add(-24 * time.Hour)},
		&fakeInstance{ID: "c", Name: "c", Created: time.Now().Add(-24 * time.Hour)},
		&fakeInstance{ID: "d", Name: "d", Created: time.Now()},
		&fakeInstance{ID: "e", Name: "e", Created: time.Now().Add(-24 * time.Hour)},
		&fakeInstance{ID: "f", Name: "f", Created: time.Now()},
	}

	nodes := []*core_v1.Node{
		&core_v1.Node{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: "e",
			},
		},
		&core_v1.Node{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: "f",
			},
		},
	}

	client := &MockKlusterClient{}
	client.On("DeleteNode", "a").Return(nil)
	client.On("DeleteNode", "b").Return(fmt.Errorf("Boom"))
	client.On("DeleteNode", "c").Return(nil)
	client.On("DeleteNode", "d").Return(nil)
	client.On("DeleteNode", "e").Return(nil)
	client.On("DeleteNode", "f").Return(nil)

	reconciler := flightReconciler{
		kluster,
		instances,
		nodes,
		client,
		log.NewNopLogger(),
	}

	ids := reconciler.DeleteIncompletelySpawnedInstances()
	client.AssertCalled(t, "DeleteNode", "a")
	client.AssertCalled(t, "DeleteNode", "b")
	client.AssertCalled(t, "DeleteNode", "c")
	client.AssertNotCalled(t, "DeleteNode", "d")
	client.AssertNotCalled(t, "DeleteNode", "e")
	client.AssertNotCalled(t, "DeleteNode", "f")
	assert.ElementsMatch(t, ids, []string{"a", "c"})
}

func TestEnsureKubernikusRuleInSecurityGroup(t *testing.T) {
	kluster := &v1.Kluster{}
	instances := []Instance{}
	nodes := []*core_v1.Node{}

	client := &MockKlusterClient{}
	client.On("EnsureKubernikusRuleInSecurityGroup").Return(true, nil)

	reconciler := flightReconciler{
		kluster,
		instances,
		nodes,
		client,
		log.NewNopLogger(),
	}

	ensured := reconciler.EnsureKubernikusRuleInSecurityGroup()
	client.AssertCalled(t, "EnsureKubernikusRuleInSecurityGroup")
	assert.True(t, ensured)
}
