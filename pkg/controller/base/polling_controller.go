package base

import (
	"sync"
	"time"

	"github.com/go-kit/log"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	informers_v1 "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
)

type PollingReconciler interface {
	Reconcile(kluster *v1.Kluster) error
}

type pollingController struct {
	logger       log.Logger
	watchers     sync.Map
	syncPeriod   time.Duration
	klusterIndex cache.Indexer
	reconciler   PollingReconciler
}

func NewPollingController(syncPeriod time.Duration, informer informers_v1.KlusterInformer, reconciler PollingReconciler, logger log.Logger) Controller {
	controller := &pollingController{
		logger:       logger,
		syncPeriod:   syncPeriod,
		klusterIndex: informer.Informer().GetIndexer(),
		reconciler:   reconciler,
	}

	informer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.klusterAdd,
			UpdateFunc: controller.klusterUpdated,
			DeleteFunc: controller.klusterDelete,
		})
	return controller
}

func (pc *pollingController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	pc.logger.Log("msg", "Starting", "interval", pc.syncPeriod)
	defer pc.logger.Log("msg", "Stopped")
	<-stopCh
	//Stop all reconciliation loops
	pc.watchers.Range(func(key, value interface{}) bool {
		close(value.(chan struct{}))
		return true
	})
}

func (pc *pollingController) klusterAdd(obj interface{}) {
	if obj.(*v1.Kluster).Disabled() {
		//stop the watcher for disabled klusters
		pc.klusterDelete(obj)
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	closeCh := make(chan struct{})
	if _, alreadyStored := pc.watchers.LoadOrStore(key, closeCh); alreadyStored {
		return
	}
	go pc.watchKluster(key, closeCh)
}
func (p *pollingController) klusterUpdated(_, obj interface{}) {
	p.klusterAdd(obj)
}

func (pc *pollingController) watchKluster(key string, stop <-chan struct{}) {
	reconcile := func() {
		obj, exists, err := pc.klusterIndex.GetByKey(key)
		if !exists || err != nil {
			return
		}
		kluster := obj.(*v1.Kluster)
		logger := log.With(pc.logger, "kluster", kluster.Name)
		begin := time.Now()
		err = pc.reconciler.Reconcile(kluster)
		logger.Log("msg", "Reconciling", "took", time.Since(begin), "v", 5, "err", err)
	}
	wait.JitterUntil(reconcile, pc.syncPeriod, 0.5, true, stop)
}

func (pc *pollingController) klusterDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	if stopCh, found := pc.watchers.Load(key); found {
		close(stopCh.(chan struct{}))
		pc.watchers.Delete(key)
	}
}
