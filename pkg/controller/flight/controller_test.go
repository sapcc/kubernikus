package flight

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type MockFlightReconcilerFactory struct {
	mock.Mock
}

type MockFlightReconciler struct {
	mock.Mock
}

func (m *MockFlightReconcilerFactory) FlightReconciler(kluster *v1.Kluster) (FlightReconciler, error) {
	args := m.Called(kluster)
	return args.Get(0).(FlightReconciler), args.Error(1)
}

func (m *MockFlightReconciler) EnsureInstanceSecurityGroupAssignment() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockFlightReconciler) EnsureKubernikusRulesInSecurityGroup() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockFlightReconciler) DeleteIncompletelySpawnedInstances() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockFlightReconciler) DeleteErroredInstances() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockFlightReconciler) EnsureServiceUserRoles() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockFlightReconciler) EnsureNodeMetadataAndTags() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestReconcile(t *testing.T) {
	kluster := &v1.Kluster{}
	kluster.Status.Phase = models.KlusterPhaseRunning

	reconciler := &MockFlightReconciler{}
	reconciler.On("EnsureKubernikusRulesInSecurityGroup").Return(true)
	reconciler.On("EnsureInstanceSecurityGroupAssignment").Return([]string{})
	reconciler.On("DeleteIncompletelySpawnedInstances").Return([]string{})
	reconciler.On("DeleteErroredInstances").Return([]string{})
	reconciler.On("EnsureServiceUserRoles").Return([]string{})
	reconciler.On("EnsureNodeMetadataAndTags").Return([]string{})

	factory := &MockFlightReconcilerFactory{}
	factory.On("FlightReconciler", kluster).Return(reconciler, nil)

	controller := &FlightController{factory, nil}

	_, err := controller.Reconcile(kluster)
	assert.NoError(t, err)
	factory.AssertCalled(t, "FlightReconciler", kluster)
	reconciler.AssertCalled(t, "EnsureKubernikusRulesInSecurityGroup")
	reconciler.AssertCalled(t, "EnsureInstanceSecurityGroupAssignment")
	reconciler.AssertCalled(t, "DeleteIncompletelySpawnedInstances")
	reconciler.AssertCalled(t, "DeleteErroredInstances")
	reconciler.AssertCalled(t, "EnsureServiceUserRoles")
	reconciler.AssertCalled(t, "EnsureNodeMetadataAndTags")

}
