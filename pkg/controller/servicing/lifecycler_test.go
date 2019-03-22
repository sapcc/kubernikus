package servicing

import (
	"github.com/stretchr/testify/mock"
	core_v1 "k8s.io/api/core/v1"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type MockLifeCycler struct {
	mock.Mock
}

func (m *MockLifeCycler) Drain(node *core_v1.Node) error {
	return m.Called(node).Error(0)
}

func (m *MockLifeCycler) Reboot(node *core_v1.Node) error {
	return m.Called(node).Error(0)
}

func (m *MockLifeCycler) Replace(node *core_v1.Node) error {
	return m.Called(node).Error(0)
}

type MockLifeCyclerFactory struct {
	mock.Mock
}

func (m *MockLifeCyclerFactory) Make(k *v1.Kluster) (LifeCycler, error) {
	return m.Called(k).Get(0).(LifeCycler), m.Called(k).Error(1)
}
