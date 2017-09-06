package controller

import (
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type LaunchControl struct {
	Factories
	queue workqueue.RateLimitingInterface
}

func NewLaunchController(factories Factories) *LaunchControl {
	launchctl := &LaunchControl{
		Factories: factories,
		queue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	launchctl.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	return launchctl
}

func (launchctl *LaunchControl) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer launchctl.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	glog.Infof("LaunchControl started!")

	for i := 0; i < threadiness; i++ {
		go wait.Until(launchctl.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (launchctl *LaunchControl) runWorker() {
	for launchctl.processNextWorkItem() {
	}
}

func (launchctl *LaunchControl) processNextWorkItem() bool {
	key, quit := launchctl.queue.Get()
	if quit {
		return false
	}
	defer launchctl.queue.Done(key)

	err := launchctl.handler(key.(string))
	if err == nil {
		launchctl.queue.Forget(key)
		return true
	}

	glog.Warningf("Error running handler: %v", err)
	launchctl.queue.AddRateLimited(key)

	return true
}

func (launchctl *LaunchControl) handler(key string) error {
	return nil
}
