package launch

import (
	"github.com/go-kit/kit/log"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
	informers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util"
)

const (
	LaunchctlFinalizer = "launchctl"
)

type LaunchReconciler struct {
	config.Clients

	Recorder record.EventRecorder
	Logger   log.Logger

	klusterInformer informers_kubernikus.KlusterInformer
	nodeObervatory  *nodeobservatory.NodeObservatory
}

func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger,
		"controller", "launch")

	var reconciler base.Reconciler
	reconciler = &LaunchReconciler{clients, recorder, logger, factories.Kubernikus.Kubernikus().V1().Klusters(), factories.NodesObservatory.NodeInformer()}
	reconciler = &base.LoggingReconciler{reconciler, logger}
	reconciler = &base.InstrumentingReconciler{
		reconciler,
		metrics.LaunchOperationsLatency,
		metrics.LaunchOperationsTotal,
		metrics.LaunchSuccessfulOperationsTotal,
		metrics.LaunchFailedOperationsTotal,
	}

	queue := workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(base.BASE_DELAY, base.MAX_DELAY))
	factories.NodesObservatory.NodeInformer().AddEventHandlerFuncs(nodeobservatory.NodeEventHandlerFuncs{
		AddFunc: func(kluster *v1.Kluster, node *core_v1.Node) {
			if key, err := cache.MetaNamespaceKeyFunc(kluster); err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(kluster *v1.Kluster, old, new *core_v1.Node) {
			if key, err := cache.MetaNamespaceKeyFunc(kluster); err == nil {
				if util.IsNodeReady(old) != util.IsNodeReady(new) || old.Spec.Unschedulable != new.Spec.Unschedulable {
					queue.Add(key)
				}
			}
		},
		DeleteFunc: func(kluster *v1.Kluster, node *core_v1.Node) {
			if key, err := cache.MetaNamespaceKeyFunc(kluster); err == nil {
				queue.Add(key)
			}
		},
	})

	return base.NewController(threadiness, factories, reconciler, logger, queue)
}

func (lr *LaunchReconciler) Reconcile(kluster *v1.Kluster) (requeue bool, err error) {
	switch kluster.Status.Phase {
	case models.KlusterPhaseCreating:
		util.EnsureFinalizerCreated(lr.Kubernikus.Kubernikus(), lr.klusterInformer.Lister(), kluster, LaunchctlFinalizer)
	case models.KlusterPhaseRunning:
		return lr.reconcilePools(kluster)
	case models.KlusterPhaseTerminating:
		return lr.terminatePools(kluster)
	}

	return false, nil
}

func (lr *LaunchReconciler) reconcilePools(kluster *v1.Kluster) (requeue bool, err error) {
	for _, pool := range kluster.Spec.NodePools {
		_, requeue, err = lr.reconcilePool(kluster, &pool)
		if err != nil {
			return
		}
	}

	return
}

func (lr *LaunchReconciler) terminatePools(kluster *v1.Kluster) (requeue bool, err error) {
	for _, pool := range kluster.Spec.NodePools {
		_, requeue, err = lr.terminatePool(kluster, &pool)
		if err != nil {
			return
		}
	}

	util.EnsureFinalizerRemoved(lr.Kubernikus.KubernikusV1(), lr.klusterInformer.Lister(), kluster, LaunchctlFinalizer)

	return
}

func (lr *LaunchReconciler) terminatePool(kluster *v1.Kluster, pool *models.NodePool) (status *PoolStatus, requeue bool, err error) {
	pm := lr.newPoolManager(kluster, pool)
	status, err = pm.GetStatus()
	if err != nil {
		return
	}

	for _, node := range status.Nodes {
		requeue = true
		if err = pm.DeleteNode(node); err != nil {
			return
		}
	}

	err = pm.SetStatus(status)
	return
}

func (lr *LaunchReconciler) reconcilePool(kluster *v1.Kluster, pool *models.NodePool) (status *PoolStatus, requeue bool, err error) {
	pm := lr.newPoolManager(kluster, pool)
	status, err = pm.GetStatus()
	if err != nil {
		return
	}

	switch {
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
