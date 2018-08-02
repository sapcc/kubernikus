package flight

import (
	"time"

	"github.com/go-kit/kit/log"
	core_v1 "k8s.io/api/core/v1"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	openstack_kluster "github.com/sapcc/kubernikus/pkg/client/openstack/kluster"
)

const (
	INSTANCE_SPAWNING_TIMEOUT = 25 * time.Minute
)

type FlightReconciler interface {
	EnsureInstanceSecurityGroupAssignment() []string
	DeleteIncompletelySpawnedInstances() []string
	DeleteErroredInstances() []string
	EnsureKubernikusRuleInSecurityGroup() bool
}

type flightReconciler struct {
	Kluster   *v1.Kluster
	Instances []Instance
	Nodes     []*core_v1.Node
	Client    openstack_kluster.KlusterClient
	Logger    log.Logger
}

func (f *flightReconciler) EnsureInstanceSecurityGroupAssignment() []string {
	ids := []string{}
	for _, instance := range f.Instances {
		found := false
		for _, sgn := range instance.GetSecurityGroupNames() {
			if sgn == f.Kluster.Spec.Openstack.SecurityGroupName {
				found = true
				break
			}
		}

		if found {
			continue
		}

		if err := f.Client.SetSecurityGroup(instance.GetID()); err != nil {
			f.Logger.Log(
				"msg", "couldn't set securitygroup",
				"instance", instance.GetID(),
				"err", err)
			continue
		}
		ids = append(ids, instance.GetID())
	}
	return ids
}

func (f *flightReconciler) EnsureKubernikusRuleInSecurityGroup() bool {
	ensured, err := f.Client.EnsureKubernikusRuleInSecurityGroup()
	if err != nil {
		f.Logger.Log(
			"msg", "couldn't ensure security group rules",
			"err", err)
	}
	return ensured
}

func (f *flightReconciler) DeleteIncompletelySpawnedInstances() []string {
	deletedInstanceIDs := []string{}
	timedOutInstances := f.getTimedOutInstances()
	unregisteredInstances := f.getUnregisteredInstances()

	for _, unregistered := range unregisteredInstances {
		found := false
		for _, timedOut := range timedOutInstances {
			if unregistered.GetName() == timedOut.GetName() {
				found = true
				break
			}
		}

		if found {
			if err := f.Client.DeleteNode(unregistered.GetID()); err != nil {
				f.Logger.Log(
					"msg", "couldn't delete incompletely spawned instance",
					"instance", unregistered.GetID(),
					"err", err)
				continue
			}
			deletedInstanceIDs = append(deletedInstanceIDs, unregistered.GetID())
		}
	}

	return deletedInstanceIDs
}

func (f *flightReconciler) DeleteErroredInstances() []string {
	deletedInstanceIDs := []string{}
	erroredInstances := f.getErroredInstances()

	for _, errored := range erroredInstances {
		if err := f.Client.DeleteNode(errored.GetID()); err != nil {
			f.Logger.Log(
				"msg", "couldn't delete errored instance",
				"instance", errored.GetID(),
				"err", err)
			continue
		}
		deletedInstanceIDs = append(deletedInstanceIDs, errored.GetID())
	}

	return deletedInstanceIDs
}

func (f *flightReconciler) getErroredInstances() []Instance {
	errored := []Instance{}
	for _, instance := range f.Instances {
		if instance.Erroring() {
			errored = append(errored, instance)
		}
	}
	return errored
}

func (f *flightReconciler) getTimedOutInstances() []Instance {
	timedOut := []Instance{}
	for _, instance := range f.Instances {
		if instance.GetCreated().Before(time.Now().Add(-INSTANCE_SPAWNING_TIMEOUT)) {
			timedOut = append(timedOut, instance)
		}
	}
	return timedOut
}

func (f *flightReconciler) getUnregisteredInstances() []Instance {
	unregisterd := []Instance{}
	for _, instance := range f.Instances {
		found := false
		for _, node := range f.Nodes {
			if node.GetName() == instance.GetName() {
				found = true
				break
			}
		}
		if !found {
			unregisterd = append(unregisterd, instance)
		}
	}
	return unregisterd
}
