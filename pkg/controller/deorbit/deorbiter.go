package deorbit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/util"
)

type SelfDestructReason string

const (
	APIUnavailable SelfDestructReason = "APIUnavailable"
	DeorbitHanging SelfDestructReason = "DeorbitHanging"

	// If the customer's apiserver is unreachable for this duration, we assume it is
	// already decommissioned, permanently damaged or was never created successfully
	// in the first place.
	APIUnavailableTimeout = 2 * time.Minute

	// After this duration, the deorbiter will utimately give up retrying. This will
	// potentially leave debris in the customer's project in the form of volumes,
	// load balancers or floating IPs.
	DeorbitHangingTimeout = 24 * time.Hour

	// < 1.17 clusters don't user finalizers on service objects for loadbalancer cleanup
	// We wait 5 minutes staically instead
	ServiceDeletionGracePeriod = 5 * time.Minute

	// While waiting for deletion use this interval for rechecks
	PollInterval = 15 * time.Second
)

// Snapshot group version resource for dynamic kubernetes client
var SnapshotGvr = schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots"}

type Deorbiter interface {
	DeleteSnapshots() ([]*unstructured.Unstructured, error)
	DeletePersistentVolumeClaims() ([]core_v1.PersistentVolumeClaim, error)
	DeleteServices() ([]core_v1.Service, error)
	WaitForSnapshotCleanUp() error
	WaitForPersistentVolumeCleanup() error
	WaitForServiceCleanup() error
	SelfDestruct(SelfDestructReason) error
	IsAPIUnavailableTimeout() bool
	IsDeorbitHangingTimeout() bool
}

type ConcreteDeorbiter struct {
	Kluster       *v1.Kluster
	Stop          <-chan struct{}
	Client        kubernetes.Interface
	DynamicClient dynamic.Interface
	Logger        log.Logger
	ServiceClient *gophercloud.ServiceClient
}

func NewDeorbiter(kluster *v1.Kluster, stopCh <-chan struct{}, clients config.Clients, recorder record.EventRecorder, logger log.Logger, serviceClient *gophercloud.ServiceClient) (Deorbiter, error) {
	client, err := clients.Satellites.ClientFor(kluster)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := clients.Satellites.DynamicClientFor(kluster)
	if err != nil {
		return nil, err
	}

	var deorbiter Deorbiter
	deorbiter = &ConcreteDeorbiter{kluster, stopCh, client, dynamicClient, logger, serviceClient}
	deorbiter = &LoggingDeorbiter{deorbiter, logger}
	deorbiter = &EventingDeorbiter{deorbiter, kluster, recorder}
	deorbiter = &InstrumentingDeorbiter{
		deorbiter,
		metrics.DeorbitOperationsLatency,
		metrics.DeorbitOperationsTotal,
		metrics.DeorbitSuccessfulOperationsTotal,
		metrics.DeorbitFailedOperationsTotal,
	}

	return deorbiter, nil
}

func (d *ConcreteDeorbiter) DeleteSnapshots() (deleted []*unstructured.Unstructured, err error) {
	deleted = []*unstructured.Unstructured{}

	snapshotList, err := d.DynamicClient.Resource(SnapshotGvr).Namespace(meta_v1.NamespaceAll).List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return deleted, err
	}

	for index := range snapshotList.Items {
		err = d.DynamicClient.Resource(SnapshotGvr).Namespace(snapshotList.Items[index].GetNamespace()).Delete(context.Background(), snapshotList.Items[index].GetName(), meta_v1.DeleteOptions{})
		if err != nil {
			return deleted, err
		}
		deleted = append(deleted, &snapshotList.Items[index])
	}
	return deleted, err
}

func (d *ConcreteDeorbiter) DeletePersistentVolumeClaims() (deleted []core_v1.PersistentVolumeClaim, err error) {
	deleted = []core_v1.PersistentVolumeClaim{}

	pvcs, err := d.Client.CoreV1().PersistentVolumeClaims(meta_v1.NamespaceAll).List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return deleted, err
	}

	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase != core_v1.ClaimBound || pvc.Spec.VolumeName == "" {
			continue
		}

		pv, err := d.Client.CoreV1().PersistentVolumes().Get(context.TODO(), pvc.Spec.VolumeName, meta_v1.GetOptions{})
		if err != nil {
			return deleted, err
		}

		if pv.Spec.Cinder == nil && pv.Spec.CSI == nil {
			continue
		}
		deleted = append(deleted, pvc)

		err = d.Client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.TODO(), pvc.Name, meta_v1.DeleteOptions{})
		if err != nil {
			return deleted, err
		}
	}

	return deleted, err
}

func (d *ConcreteDeorbiter) DeleteServices() (deleted []core_v1.Service, err error) {
	deleted = []core_v1.Service{}

	services, err := d.Client.CoreV1().Services(meta_v1.NamespaceAll).List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return deleted, err
	}

	for _, service := range services.Items {
		if service.Spec.Type != "LoadBalancer" {
			continue
		}
		deleted = append(deleted, service)

		err = d.Client.CoreV1().Services(service.Namespace).Delete(context.TODO(), service.Name, meta_v1.DeleteOptions{})
		if err != nil {
			return deleted, err
		}
	}

	return deleted, err
}

func (d *ConcreteDeorbiter) WaitForSnapshotCleanUp() (err error) {
	return wait.PollImmediateUntil(PollInterval, d.isSnapshotCleanupFinished, d.Stop) //nolint:staticcheck
}

func (d *ConcreteDeorbiter) WaitForPersistentVolumeCleanup() (err error) {
	return wait.PollImmediateUntil(PollInterval, d.isPersistentVolumesCleanupFinished, d.Stop) //nolint:staticcheck
}

func (d *ConcreteDeorbiter) WaitForServiceCleanup() (err error) {
	return wait.PollImmediateUntil(PollInterval, d.isServiceCleanupFinished, d.Stop) //nolint:staticcheck
}

func (d *ConcreteDeorbiter) SelfDestruct(reason SelfDestructReason) (err error) {
	// Self-Destruct ironically does nothing
	return nil
}

func (d *ConcreteDeorbiter) IsAPIUnavailableTimeout() bool {
	return d.Kluster.ObjectMeta.DeletionTimestamp.Add(APIUnavailableTimeout).Before(time.Now())
}

func (d *ConcreteDeorbiter) IsDeorbitHangingTimeout() bool {
	return d.Kluster.ObjectMeta.DeletionTimestamp.Add(DeorbitHangingTimeout).Before(time.Now())
}

func (d *ConcreteDeorbiter) isSnapshotCleanupFinished() (bool, error) {
	snapshots, err := d.DynamicClient.Resource(SnapshotGvr).Namespace(meta_v1.NamespaceAll).List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return false, err
	}
	if len(snapshots.Items) > 0 {
		return false, nil
	}
	return true, nil
}

func (d *ConcreteDeorbiter) isPersistentVolumesCleanupFinished() (bool, error) {
	pvs, err := d.Client.CoreV1().PersistentVolumes().List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return false, err
	}

	volumeListOpts := volumes.ListOpts{
		TenantID: d.Kluster.Account(),
	}

	allPages, err := volumes.List(d.ServiceClient, volumeListOpts).AllPages()
	if err != nil {
		return false, fmt.Errorf("There should be no error while retrieving volume pages: %v", err)
	}

	allVolumes, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		return false, fmt.Errorf("There should be no error while extracting volumes: %v", err)
	}

	for _, pv := range pvs.Items {
		// ignore failed PVs
		if pv.Status.Phase == core_v1.VolumeFailed {
			continue
		}

		// ignore volumes already deleted in openstack
		for _, volume := range allVolumes {
			if (pv.Spec.PersistentVolumeSource.Cinder != nil && pv.Spec.PersistentVolumeSource.Cinder.VolumeID == volume.ID) ||
				(pv.Spec.PersistentVolumeSource.CSI != nil && pv.Spec.PersistentVolumeSource.CSI.VolumeHandle == volume.ID) {
				return false, nil
			}
		}
	}

	return true, nil
}

func (d *ConcreteDeorbiter) isServiceCleanupFinished() (bool, error) {
	if ok, _ := util.KlusterVersionConstraint(d.Kluster, "< 1.17"); ok {
		return d.Kluster.ObjectMeta.DeletionTimestamp.Add(ServiceDeletionGracePeriod).Before(time.Now()), nil
	}
	services, err := d.Client.CoreV1().Services(meta_v1.NamespaceAll).List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, service := range services.Items {
		if service.Spec.Type != "LoadBalancer" {
			continue
		}
		return false, nil
	}
	return true, nil
}
