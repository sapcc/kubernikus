package controller

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
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
