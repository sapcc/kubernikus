package launch

import (
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"

	"github.com/go-kit/kit/log"
	"k8s.io/client-go/tools/record"
)

type LaunchReconciler struct {
	config.Clients

	Recorder record.EventRecorder
	Logger   log.Logger
}

func NewController(factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger,
		"controller", "launch")

	var reconciler base.Reconciler
	reconciler = &LaunchReconciler{clients, recorder, logger}
	reconciler = &base.LoggingReconciler{reconciler, logger}
	reconciler = &base.InstrumentingReconciler{
		reconciler,
		metrics.LaunchOperationsLatency,
		metrics.LaunchOperationsTotal,
		metrics.LaunchSuccessfulOperationsTotal,
		metrics.LaunchFailedOperationsTotal,
	}

	return base.NewController(factories, clients, reconciler, logger)
}

func (lr *LaunchReconciler) Reconcile(kluster *v1.Kluster) (requeueRequested bool, err error) {
	if !(kluster.Status.Phase == models.KlusterPhaseRunning || kluster.Status.Phase == models.KlusterPhaseTerminating) {
		return false, nil
	}

	for _, pool := range kluster.Spec.NodePools {
		_, requeue, err := lr.reconcilePool(kluster, &pool)
		if err != nil {
			return false, err
		}

		if requeue {
			requeueRequested = true
		}
	}

	return requeueRequested, nil
}

func (lr *LaunchReconciler) reconcilePool(kluster *v1.Kluster, pool *models.NodePool) (status *PoolStatus, requeue bool, err error) {
	pm := lr.newPoolManager(kluster, pool)
	status, err = pm.GetStatus()
	if err != nil {
		return
	}

	switch {
	case kluster.Status.Phase == models.KlusterPhaseTerminating:
		for _, node := range status.Nodes {
			requeue = true
			if err = pm.DeleteNode(node); err != nil {
				return
			}
		}
		return
	case status.Needed > 0:
		for i := 0; i < int(status.Needed); i++ {
			requeue = true
			if _, err = pm.CreateNode(); err != nil {
				return
			}
		}
		return
	case status.UnNeeded > 0:
		for i := 0; i < int(status.UnNeeded); i++ {
			requeue = true
			if err = pm.DeleteNode(status.Nodes[i]); err != nil {
				return
			}
		}
		return
	case status.Starting > 0:
		requeue = true
	case status.Stopping > 0:
		requeue = true
	}

	err = pm.SetStatus(status)
	return
}
