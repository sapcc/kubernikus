package deorbit

import (
	"fmt"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type FakeDeorbiter struct {
	CinderPVCCount int
	LBServiceCount int
	SnapshotCount  int

	APIDown bool
	Hanging bool

	HasCalledDeleteSnapshots                bool
	HasCalledDeletePersistentVolumeClaims   bool
	HasCalledDeleteServices                 bool
	HasCalledWaitForSnapshotCleanup         bool
	HasCalledWaitForPersistentVolumeCleanup bool
	HasCalledWaitForServiceCleanup          bool
	HasCalledSeldDestruct                   bool

	SelfDestructReason SelfDestructReason
}

func (d *FakeDeorbiter) DeleteSnapshots() (deleted []*unstructured.Unstructured, err error) {
	d.HasCalledDeleteSnapshots = true

	for i := 0; i < d.SnapshotCount; i++ {
		snapshot := &unstructured.Unstructured{}
		snapshot.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "snapshot.storage.k8s.io/v1",
			"kind":       "VolumeSnapshot",
			"metadata": map[string]interface{}{
				"name": fmt.Sprintf("volume-snapshot-%d", i),
			},
			"spec": map[string]interface{}{
				"volumeSnapshotClassName": "csi-cinder-snapclass",
				"source": map[string]interface{}{
					"persistentVolumeClaimName": "pvc-hostname",
				},
			},
		})
		deleted = append(deleted, snapshot)
	}

	return deleted, nil
}

func (d *FakeDeorbiter) DeletePersistentVolumeClaims() (deleted []core_v1.PersistentVolumeClaim, err error) {
	d.HasCalledDeletePersistentVolumeClaims = true

	for i := 0; i < d.CinderPVCCount; i++ {
		deleted = append(deleted,
			core_v1.PersistentVolumeClaim{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: fmt.Sprintf("pvc-cinder%d", i),
				},
				Spec: core_v1.PersistentVolumeClaimSpec{
					VolumeName: fmt.Sprintf("pv-cinder%d", i),
				},
			},
		)
	}

	return deleted, nil
}

func (d *FakeDeorbiter) DeleteServices() (deleted []core_v1.Service, err error) {
	d.HasCalledDeleteServices = true

	for i := 0; i < d.LBServiceCount; i++ {
		deleted = append(deleted,
			core_v1.Service{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: fmt.Sprintf("svc-lb%d", i),
				},
				Spec: core_v1.ServiceSpec{
					Type: "LoadBalancer",
				},
			},
		)
	}

	return deleted, nil
}

func (d *FakeDeorbiter) WaitForSnapshotCleanUp() (err error) {
	d.HasCalledWaitForSnapshotCleanup = true
	return nil
}

func (d *FakeDeorbiter) WaitForPersistentVolumeCleanup() (err error) {
	d.HasCalledWaitForPersistentVolumeCleanup = true

	return nil
}

func (d *FakeDeorbiter) WaitForServiceCleanup() (err error) {
	d.HasCalledWaitForServiceCleanup = true

	return nil
}

func (d *FakeDeorbiter) SelfDestruct(reason SelfDestructReason) (err error) {
	d.HasCalledSeldDestruct = true
	d.SelfDestructReason = reason
	return nil
}

func (d *FakeDeorbiter) IsAPIUnavailableTimeout() bool {
	return d.APIDown
}

func (d *FakeDeorbiter) IsDeorbitHangingTimeout() bool {
	return d.Hanging
}
