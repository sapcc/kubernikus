package controller

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller interface {
	Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup)
}

type BaseController interface {
	Controller
	reconcile(key string) error
}

type Base struct {
	Clients
	queue      workqueue.RateLimitingInterface
	informer   cache.SharedIndexInformer
	Controller BaseController
}

func NewBaseController(clients Clients, informer cache.SharedIndexInformer) Base {
	base := Base{
		Clients:  clients,
		queue:    workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		informer: informer,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				base.queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				base.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				base.queue.Add(key)
			}
		},
	})

	return base
}

func (base *Base) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer base.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	glog.Infof("Starting %v with %d workers", base.getName(), threadiness)

	for i := 0; i < threadiness; i++ {
		go wait.Until(base.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(KLUSTER_RECHECK_INTERVAL)
	go func() {
		for {
			select {
			case <-ticker.C:
				for key := range base.informer.GetStore().ListKeys() {
					base.queue.Add(key)
				}
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	<-stopCh
}

func (base *Base) runWorker() {
	for base.processNextWorkItem() {
	}
}

func (base *Base) processNextWorkItem() bool {
	key, quit := base.queue.Get()
	if quit {
		return false
	}
	defer base.queue.Done(key)

	// Invoke the method containing the business logic
	err := base.reconciliation(key.(string))
	base.handleErr(err, key)
	return true
}

func (base *Base) reconcile(key string) error {
	return fmt.Errorf("NotImplemented")
}

func (base *Base) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		base.queue.Forget(key)
		return
	}

	glog.Infof("[%v] Error while processing %v: %v", base.getName(), key, err)
	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if base.queue.NumRequeues(key) < 5 {
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		base.queue.AddRateLimited(key)
		return
	}

	glog.V(5).Infof("[%v] Dropping %v from queue because of too many errors...", base.getName(), key)
	base.queue.Forget(key)
}

func getControllerName(c Controller) string {
	return reflect.TypeOf(c).Elem().Name()

}

func (base *Base) getName() string {
	return getControllerName(base.Controller)
}

func (base *Base) reconciliation(key string) error {
	glog.V(5).Infof("[%v] Reconciling %v", base.getName(), key)
	return base.Controller.reconcile(key)
}
