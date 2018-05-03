package deorbit

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
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
}

func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger, "controller", "deorbit")

	var reconciler base.Reconciler
	reconciler = &DeorbitReconciler{clients, recorder, logger, factories.Kubernikus.Kubernikus().V1().Klusters()}
	reconciler = &base.LoggingReconciler{reconciler, logger}
	reconciler = &base.InstrumentingReconciler{
		reconciler,
		metrics.DeorbitOperationsLatency,
		metrics.DeorbitOperationsTotal,
		metrics.DeorbitSuccessfulOperationsTotal,
		metrics.DeorbitFailedOperationsTotal,
	}
	return base.NewController(threadiness, factories, reconciler, logger, nil)

}

func (d *DeorbitReconciler) Reconcile(kluster *v1.Kluster) (bool, error) {
	switch kluster.Status.Phase {
	case models.KlusterPhaseRunning:
		return false, util.EnsureFinalizerCreated(d.Kubernikus.KubernikusV1(), d.klusterInformer.Lister(), kluster, DeorbiterFinalizer)
	case models.KlusterPhaseTerminating:
		if kluster.HasFinalizer(DeorbiterFinalizer) {
			if err := d.deorbit(kluster); err != nil {
				return false, err
			}
			return false, util.EnsureFinalizerRemoved(d.Kubernikus.KubernikusV1(), d.klusterInformer.Lister(), kluster, DeorbiterFinalizer)
		}
	}
	return false, nil
}

func (d *DeorbitReconciler) deorbit(kluster *v1.Kluster) (err error) {
	// The following channel is used to abort the deorbiting after a certain
	// time. It is required because the deorbit wait-functions block indefinitely or
	// until the stop channel is closed. This is used to unblock the workqueue.
	done := make(chan struct{})

	timer := time.NewTimer(UnblockWorkerTimeout)
	defer timer.Stop()

	go func() {
		<-timer.C
		d.Logger.Log("msg", "timeout waiting. unblocking the worker routine")
		close(done)
	}()

	deorbiter, err := NewDeorbiter(kluster, done, d.Clients, d.Recorder, d.Logger)
	if err != nil {
		return err
	}

	err = d.doDeorbit(deorbiter)

	return d.doSelfDestruct(deorbiter, err)
}

func (d *DeorbitReconciler) doDeorbit(deorbiter Deorbiter) (err error) {
	deletedPVCs, err := deorbiter.DeletePersistentVolumeClaims()
	if err != nil {
		return err
	}

	deletedServices, err := deorbiter.DeleteServices()
	if err != nil {
		return err
	}

	if len(deletedPVCs) > 0 {
		if err := deorbiter.WaitForPersistentVolumeCleanup(); err != nil {
			return err
		}
	}

	if len(deletedServices) > 0 {
		if err := deorbiter.WaitForServiceCleanup(); err != nil {
			return err
		}
	}

	return nil
}

func (d *DeorbitReconciler) doSelfDestruct(deorbiter Deorbiter, outer error) (err error) {
	// If for some reason communication with the Kluster's apiserver is not possible,
	// we retry until a timeout is reached. Then self-destruct and accept debris.
	if errors.IsUnexpectedServerError(outer) || errors.IsServerTimeout(outer) {
		if deorbiter.IsAPIUnavailableTimeout() {
			err = deorbiter.SelfDestruct(APIUnavailable)
		}
	}

	// Self-Destruct if the Kluster is stuck in deorbitting for a long time. This
	// timeout is long enough for reaction of an operations team and alerts to fire.
	// Keeps the chance to analyse what went wrong. Eventually this unstuckes the
	// Kluster automatically without human interaction. It frees up the Kluster with
	// the downside of potential debris in the customer's project.
	if deorbiter.IsDeorbitHangingTimeout() {
		err = deorbiter.SelfDestruct(DeorbitHanging)
	}

	return err
}
