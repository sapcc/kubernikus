package deorbit

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	informers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util"
)

func init() {
	prometheus.MustRegister(
		metrics.DeorbitOperationsLatency,
		metrics.DeorbitOperationsTotal,
		metrics.DeorbitSuccessfulOperationsTotal,
		metrics.DeorbitFailedOperationsTotal,
	)
}

const (
	DeorbiterFinalizer = "deorbiter"

	// The longest time a single worker will be blocked, while waiting for the clean
	// up of OpenStack resources. After this timeout, deorbiting will be queued again.
	// It's a safeguard against having all workers hanging indefinitely and congesting
	// the queue.
	UnblockWorkerTimeout = 10 * time.Minute
)

type DeorbitReconciler struct {
	config.Clients

	Recorder record.EventRecorder
	Logger   log.Logger

	klusterInformer informers_kubernikus.KlusterInformer

	config.Factories
}

func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger, "controller", "deorbit")

	var reconciler base.Reconciler
	reconciler = &DeorbitReconciler{
		clients,
		recorder,
		logger,
		factories.Kubernikus.Kubernikus().V1().Klusters(),
		factories,
	}
	reconciler = &base.LoggingReconciler{Reconciler: reconciler, Logger: logger}
	reconciler = &base.InstrumentingReconciler{
		Reconciler: reconciler,
		Latency:    metrics.DeorbitOperationsLatency,
		Total:      metrics.DeorbitOperationsTotal,
		Successful: metrics.DeorbitSuccessfulOperationsTotal,
		Failed:     metrics.DeorbitFailedOperationsTotal,
	}
	return base.NewController(threadiness, factories, reconciler, logger, nil, "deorbit")

}

func (d *DeorbitReconciler) Reconcile(kluster *v1.Kluster) (bool, error) {
	switch kluster.Status.Phase {
	case models.KlusterPhaseRunning:
		return false, util.EnsureFinalizerCreated(d.Clients.Kubernikus.KubernikusV1(), d.klusterInformer.Lister(), kluster, DeorbiterFinalizer)
	case models.KlusterPhaseTerminating:
		if kluster.TerminationProtection() {
			return false, nil
		}
		if kluster.HasFinalizer(DeorbiterFinalizer) {
			if err := d.deorbit(kluster); err != nil {
				return false, err
			}
			return false, util.EnsureFinalizerRemoved(d.Clients.Kubernikus.KubernikusV1(), d.klusterInformer.Lister(), kluster, DeorbiterFinalizer)
		}
	}
	return false, nil
}

func (d *DeorbitReconciler) deorbit(kluster *v1.Kluster) (err error) {

	logger := log.With(d.Logger, "kluster", kluster.Spec.Name)
	// The following channel is used to abort the deorbiting after a certain
	// time. It is required because the deorbit wait-functions block indefinitely or
	// until the stop channel is closed. This is used to unblock the workqueue.
	done := make(chan struct{})
	var once sync.Once

	timer := time.AfterFunc(UnblockWorkerTimeout, func() {
		logger.Log("msg", "timeout waiting. unblocking the worker routine")
		once.Do(func() { close(done) })
	})
	defer timer.Stop()
	//We need to ensure the done channel is closed otherwise we leak goroutines (thanks to wait.PollUntil)
	defer once.Do(func() { close(done) })

	providerClient, err := d.Factories.Openstack.ProviderClientForKluster(kluster, logger)
	if err != nil {
		return fmt.Errorf("Could not get openstack provider client: %v", err)
	}

	serviceClient, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("Could not create block storage client: %v", err)
	}

	deorbiter, err := NewDeorbiter(kluster, done, d.Clients, d.Recorder, logger, serviceClient)
	if err != nil {
		return err
	}

	err = d.doDeorbit(deorbiter)

	return d.doSelfDestruct(deorbiter, err)
}

func (d *DeorbitReconciler) doDeorbit(deorbiter Deorbiter) (err error) {

	_, err = deorbiter.DeleteSnapshots()
	if err != nil {
		return err
	}

	_, err = deorbiter.DeletePersistentVolumeClaims()
	if err != nil {
		return err
	}

	_, err = deorbiter.DeleteServices()
	if err != nil {
		return err
	}

	if err := deorbiter.WaitForSnapshotCleanUp(); err != nil {
		return err
	}

	if err := deorbiter.WaitForPersistentVolumeCleanup(); err != nil {
		return err
	}

	if err := deorbiter.WaitForServiceCleanup(); err != nil {
		return err
	}

	return nil
}

func (d *DeorbitReconciler) doSelfDestruct(deorbiter Deorbiter, err error) error {
	// If for some reason communication with the Kluster's apiserver is not possible,
	// we retry until a timeout is reached. Then self-destruct and accept debris.
	if errors.IsUnexpectedServerError(err) || errors.IsServerTimeout(err) {
		if deorbiter.IsAPIUnavailableTimeout() {
			return deorbiter.SelfDestruct(APIUnavailable)
		}
	}

	// Self-Destruct if the Kluster is stuck in deorbitting for a long time. This
	// timeout is long enough for reaction of an operations team and alerts to fire.
	// Keeps the chance to analyse what went wrong. Eventually this unstuckes the
	// Kluster automatically without human interaction. It frees up the Kluster with
	// the downside of potential debris in the customer's project.
	if deorbiter.IsDeorbitHangingTimeout() {
		return deorbiter.SelfDestruct(DeorbitHanging)
	}

	return err
}
