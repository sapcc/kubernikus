package deorbit

import (
	"fmt"

	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/record"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/events"
)

type EventingDeorbiter struct {
	Deorbiter Deorbiter
	Kluster   *v1.Kluster
	Recorder  record.EventRecorder
}

func (d *EventingDeorbiter) DeleteSnapshots() (deletedSnapshots []*unstructured.Unstructured, err error) {
	deletedSnapshots, err = d.Deorbiter.DeleteSnapshots()

	successfullyDeletedSnapshotLength := len(deletedSnapshots)

	if err != nil && successfullyDeletedSnapshotLength > 0 {
		defer d.Recorder.Eventf(d.Kluster, core_v1.EventTypeWarning, events.FailedDeorbitSnapshot, "Failed to delete snapshots(%v): %v", deletedSnapshots[len(deletedSnapshots)-1], err)
		successfullyDeletedSnapshotLength = len(deletedSnapshots) - 1
	}

	for i := 0; i < successfullyDeletedSnapshotLength; i++ {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.SuccessfulDeorbitSnapshot, "Successfully deleted snapshots: %v", fmt.Sprintf("%v/%v", deletedSnapshots[i].GetNamespace(), deletedSnapshots[i].GetName()))
	}

	return
}

func (d *EventingDeorbiter) DeletePersistentVolumeClaims() (deletedPVCs []core_v1.PersistentVolumeClaim, err error) {
	deletedPVCs, err = d.Deorbiter.DeletePersistentVolumeClaims()

	successfullyDeletedPVCLength := len(deletedPVCs)

	if err != nil && successfullyDeletedPVCLength > 0 {
		defer d.Recorder.Eventf(d.Kluster, core_v1.EventTypeWarning, events.FailedDeorbitPVC, "Failed to delete persistent volume claims(%v): %v", deletedPVCs[len(deletedPVCs)-1], err)
		successfullyDeletedPVCLength = len(deletedPVCs) - 1
	}

	for i := 0; i < successfullyDeletedPVCLength; i++ {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.SuccessfulDeorbitPVC, "Successfully deleted persistent volume claims: %v", fmt.Sprintf("%v/%v", deletedPVCs[i].GetNamespace(), deletedPVCs[i].GetName()))
	}

	return
}

func (d *EventingDeorbiter) DeleteServices() (deletedServices []core_v1.Service, err error) {
	deletedServices, err = d.Deorbiter.DeleteServices()

	successfullyDeletedServiceLength := len(deletedServices)

	if err != nil && successfullyDeletedServiceLength > 0 {
		defer d.Recorder.Eventf(d.Kluster, core_v1.EventTypeWarning, events.FailedDeorbitService, "Failed to delete service of type LoadBalancer (%v): %v", deletedServices[len(deletedServices)-1], err)
		successfullyDeletedServiceLength = len(deletedServices) - 1
	}

	for i := 0; i < successfullyDeletedServiceLength; i++ {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.SuccessfulDeorbitService, "Successfully deleted service of type LoadBalancer : %v", fmt.Sprintf("%v/%v", deletedServices[i].GetNamespace(), deletedServices[i].GetName()))
	}

	return
}

func (d *EventingDeorbiter) WaitForSnapshotCleanUp() (err error) {
	d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.WaitingForDeorbitSnapshots, "Waiting for cleanup of Snapshots")

	err = d.Deorbiter.WaitForSnapshotCleanUp()

	if err == nil {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.SuccessfulDeorbitSnapshots, "Successfully cleaned up Snapshots")
	} else {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.FailedDeorbitSnapshots, "Failed to clean up Snapshots: %v", err)
	}

	return
}

func (d *EventingDeorbiter) WaitForPersistentVolumeCleanup() (err error) {
	d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.WaitingForDeorbitPVs, "Waiting for cleanup of Cinder volumes")

	err = d.Deorbiter.WaitForPersistentVolumeCleanup()

	if err == nil {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.SuccessfulDeorbitPVs, "Successfully cleaned up Cinder volumes")
	} else {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.FailedDeorbitPVs, "Failed to clean up Cinder volumes: %v", err)
	}

	return
}

func (d *EventingDeorbiter) WaitForServiceCleanup() (err error) {
	d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.WaitingForDeorbitLoadBalancers, "Waiting for cleanup of load balancers")

	err = d.Deorbiter.WaitForServiceCleanup()

	if err == nil {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.SuccessfulDeorbitLoadBalancers, "Successfully cleaned up load balancers")
	} else {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.FailedDeorbitLoadBalancers, "Failed to clean up load balancers: %v", err)
	}

	return
}

func (d *EventingDeorbiter) SelfDestruct(reason SelfDestructReason) (err error) {
	err = d.Deorbiter.SelfDestruct(reason)

	if err == nil {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.SuccessfulDeorbitSelfDestruct, "Failed to gracefully terminate the cluster. Initiated self-destruction. There might be left-over volumes and load balancers.")
	} else {
		d.Recorder.Eventf(d.Kluster, core_v1.EventTypeNormal, events.FailedDeorbitSelfDestruct, "Failed to activate self-destruction. There might be left-over volumes and load balancers.", err)
	}

	return
}

func (d *EventingDeorbiter) IsAPIUnavailableTimeout() bool {
	return d.Deorbiter.IsAPIUnavailableTimeout()
}

func (d *EventingDeorbiter) IsDeorbitHangingTimeout() bool {
	return d.Deorbiter.IsDeorbitHangingTimeout()
}
