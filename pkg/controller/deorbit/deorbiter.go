package deorbit

import (
	"time"

	"github.com/go-kit/kit/log"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
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

	// While waiting for deletion use this interval for rechecks
	PollInterval = 15 * time.Second
)

type Deorbiter interface {
	DeletePersistentVolumeClaims() ([]core_v1.PersistentVolumeClaim, error)
	DeleteServices() ([]core_v1.Service, error)
	WaitForPersistentVolumeCleanup() error
	WaitForServiceCleanup() error
	SelfDestruct(SelfDestructReason) error
	IsAPIUnavailableTimeout() bool
	IsDeorbitHangingTimeout() bool
}

type ConcreteDeorbiter struct {
	Kluster *v1.Kluster
	Stop    <-chan struct{}
	Client  kubernetes.Interface
	Logger  log.Logger
}

func NewDeorbiter(kluster *v1.Kluster, stopCh <-chan struct{}, clients config.Clients, recorder record.EventRecorder, logger log.Logger) (Deorbiter, error) {
	client, err := clients.Satellites.ClientFor(kluster)
	if err != nil {
		return nil, err
	}

	var deorbiter Deorbiter
	deorbiter = &ConcreteDeorbiter{kluster, stopCh, client, logger}
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

func (d *ConcreteDeorbiter) DeletePersistentVolumeClaims() (deleted []core_v1.PersistentVolumeClaim, err error) {
	deleted = []core_v1.PersistentVolumeClaim{}

	pvcs, err := d.Client.Core().PersistentVolumeClaims(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	if err != nil {
		return deleted, err
	}

	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase != core_v1.ClaimBound || pvc.Spec.VolumeName == "" {
			continue
		}

		pv, err := d.Client.Core().PersistentVolumes().Get(pvc.Spec.VolumeName, meta_v1.GetOptions{})
		if err != nil {
			return deleted, err
		}

		if pv.Spec.Cinder == nil && pv.Spec.CSI == nil {
			continue
		}
		deleted = append(deleted, pvc)

		err = d.Client.Core().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &meta_v1.DeleteOptions{})
		if err != nil {
			return deleted, err
		}
	}

	return deleted, err
}

func (d *ConcreteDeorbiter) DeleteServices() (deleted []core_v1.Service, err error) {
	deleted = []core_v1.Service{}

	services, err := d.Client.Core().Services(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	if err != nil {
		return deleted, err
	}

	for _, service := range services.Items {
		if service.Spec.Type != "LoadBalancer" {
			continue
		}
		deleted = append(deleted, service)

		err = d.Client.Core().Services(service.Namespace).Delete(service.Name, &meta_v1.DeleteOptions{})
		if err != nil {
			return deleted, err
		}
	}

	return deleted, err
}

func (d *ConcreteDeorbiter) WaitForPersistentVolumeCleanup() (err error) {
	done, err := d.isPersistentVolumesCleanupFinished()
	if err != nil {
		return err
	}

	if done {
		return nil
	}

	return wait.PollUntil(PollInterval, d.isPersistentVolumesCleanupFinished, d.Stop)
}

func (d *ConcreteDeorbiter) WaitForServiceCleanup() (err error) {
	done, err := d.isServiceCleanupFinished()
	if err != nil {
		return err
	}

	if done {
		return nil
	}

	return wait.PollUntil(PollInterval, d.isServiceCleanupFinished, d.Stop)
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

func (d *ConcreteDeorbiter) isPersistentVolumesCleanupFinished() (bool, error) {
	pvs, err := d.Client.Core().PersistentVolumes().List(meta_v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, pv := range pvs.Items {
		//ignore failed PVs
		if pv.Status.Phase == core_v1.VolumeFailed {
			continue
		}
		if pv.Spec.PersistentVolumeSource.Cinder != nil || (pv.Spec.PersistentVolumeSource.CSI != nil && pv.Spec.PersistentVolumeSource.CSI.Driver == "cinder.csi.openstack.org") {
			return false, nil
		}
	}

	return true, nil
}

func (d *ConcreteDeorbiter) isServiceCleanupFinished() (bool, error) {
	services, err := d.Client.Core().Services(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
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
