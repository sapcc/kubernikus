package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/templates"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type LaunchControl struct {
	Factories
	Clients
	queue workqueue.RateLimitingInterface
}

func NewLaunchController(factories Factories, clients Clients) *LaunchControl {
	launchctl := &LaunchControl{
		Factories: factories,
		Clients:   clients,
		queue:     workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
	}

	launchctl.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				launchctl.queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				launchctl.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				launchctl.queue.Add(key)
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

	ticker := time.NewTicker(KLUSTER_RECHECK_INTERVAL)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(2).Infof("Running periodic recheck. Queuing all Klusters...")

				klusters, err := launchctl.Factories.Kubernikus.Kubernikus().V1().Klusters().Lister().List(labels.Everything())
				if err != nil {
					glog.Errorf("Couldn't run periodic recheck. Listing klusters failed: %v", err)
				}

				for _, kluster := range klusters {
					key, err := cache.MetaNamespaceKeyFunc(kluster)
					if err == nil {
						launchctl.queue.Add(key)
					}
				}
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

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

	// Invoke the method containing the business logic
	err := launchctl.reconcile(key.(string))
	launchctl.handleErr(err, key)
	return true
}

func (launchctl *LaunchControl) requeue(kluster *v1.Kluster) {
	key, err := cache.MetaNamespaceKeyFunc(kluster)
	if err == nil {
		launchctl.queue.AddAfter(key, 5*time.Second)
	}
}

func (launchctl *LaunchControl) reconcile(key string) error {
	obj, exists, err := launchctl.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("Failed to fetch key %s from cache: %s", key, err)
	}
	if !exists {
		glog.Infof("Kluster %s deleted in the meantime", key)
		return nil
	}

	kluster := obj.(*v1.Kluster)
	glog.V(2).Infof("Handling kluster %v", kluster.Name)

	//_, err = templates.Ignition.GenerateNode(kluster, launchctl.Clients.Kubernetes)
	//if err != nil {
	//  glog.Errorf("%v", err)
	//}

	for _, pool := range kluster.Spec.NodePools {
		err := launchctl.syncPool(kluster, &pool)
		if err != nil {
			return err
		}
	}

	return nil
}

func (launchctl *LaunchControl) syncPool(kluster *v1.Kluster, pool *v1.NodePool) error {
	nodes, err := launchctl.Clients.Openstack.GetNodes(kluster, pool)
	if err != nil {
		return fmt.Errorf("Couldn't list nodes for %v/%v: %v", kluster.Name, pool.Name, err)
	}

	ready := ready(nodes)

	switch {
	case ready < pool.Size:
		glog.V(3).Infof("Pool %v/%v: Running %v/%v. Too few nodes. Need to spawn more.", kluster.Name, pool.Name, ready, pool.Size)
		return launchctl.createNode(kluster, pool)
	case ready > pool.Size:
		glog.V(3).Infof("Pool %v/%v: Running %v/%v. Too many nodes. Need to delete some.", kluster.Name, pool.Name, ready, pool.Size)
		return launchctl.terminateNode(kluster, nodes[0].ID)
	case ready == pool.Size:
		glog.V(3).Infof("Pool %v/%v: Running %v/%v. All good. Doing nothing.", kluster.Name, pool.Name, ready, pool.Size)
	}

	return nil
}

func (launchctl *LaunchControl) createNode(kluster *v1.Kluster, pool *v1.NodePool) error {
	glog.V(2).Infof("Pool %v/%v: Creating new node", kluster.Name, pool.Name)

	userdata, err := templates.Ignition.GenerateNode(kluster, launchctl.Clients.Kubernetes)
	if err != nil {
		glog.Errorf("Ignition userdata couldn't be generated: %v", err)
	}

	id, err := launchctl.Clients.Openstack.CreateNode(kluster, pool, userdata)
	if err != nil {
		return err
	}

	glog.V(2).Infof("Pool %v/%v: Created node %v.", kluster.Name, pool.Name, id)

	launchctl.requeue(kluster)
	return nil
}

func (launchctl *LaunchControl) terminateNode(kluster *v1.Kluster, id string) error {
	launchctl.requeue(kluster)
	return nil
}

func (launchctl *LaunchControl) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		launchctl.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if launchctl.queue.NumRequeues(key) < 5 {
		glog.Errorf("Error while managing nodes for kluster %q: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		launchctl.queue.AddRateLimited(key)
		return
	}

	launchctl.queue.Forget(key)
	glog.Infof("Dropping kluster %q out of the queue. Too many retries: %v", key, err)
}

func ready(nodes []openstack.Node) int {
	ready := 0
	for _, n := range nodes {
		if n.Ready() {
			ready = ready + 1
		}
	}

	return ready
}
