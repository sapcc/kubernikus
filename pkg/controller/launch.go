package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/templates"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	glog.Infof(`Starting LaunchControl with %d "threads"`, threadiness)

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
		glog.Infof("[%v] Kluster deleted in the meantime", key)
		return nil
	}

	kluster := obj.(*v1.Kluster)
	glog.V(5).Infof("[%v] Reconciling", kluster.Name)

	if !(kluster.Status.Kluster.State == v1.KlusterReady || kluster.Status.Kluster.State == v1.KlusterTerminating) {
		return fmt.Errorf("[%v] Kluster is not yet ready. Requeuing.", kluster.Name)
	}

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
		return fmt.Errorf("[%v] Couldn't list nodes for pool %v: %v", kluster.Name, pool.Name, err)
	}

	running := running(nodes)
	starting := starting(nodes)
	ready := running + starting

	info := v1.NodePoolInfo{
		Name:        pool.Name,
		Size:        pool.Size,
		Running:     running + starting, // Should be running only
		Healthy:     running,
		Schedulable: running,
	}

	if err = launchctl.updateNodePoolStatus(kluster, info); err != nil {
		return err
	}

	if kluster.Status.Kluster.State == v1.KlusterTerminating {
		if toBeTerminated(nodes) > 0 {
			glog.V(3).Infof("[%v] Kluster is terminating. Terminating Nodes for Pool %v.", kluster.Name, pool.Name)
			for _, node := range nodes {
				err := launchctl.terminateNode(kluster, node.ID)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	switch {
	case ready < pool.Size:
		glog.V(3).Infof("[%v] Pool %v: Starting/Running/Total: %v/%v/%v. Too few nodes. Need to spawn more.", kluster.Name, pool.Name, starting, running, pool.Size)
		return launchctl.createNode(kluster, pool)
	case ready > pool.Size:
		glog.V(3).Infof("[%v] Pool %v: Starting/Running/Total: %v/%v/%v. Too many nodes. Need to delete some.", kluster.Name, pool.Name, starting, running, pool.Size)
		return launchctl.terminateNode(kluster, nodes[0].ID)
	case ready == pool.Size:
		glog.V(3).Infof("[%v] Pool %v: Starting/Running/Total: %v/%v/%v. All good. Doing nothing.", kluster.Name, pool.Name, starting, running, pool.Size)
	}

	return nil
}

func (launchctl *LaunchControl) createNode(kluster *v1.Kluster, pool *v1.NodePool) error {
	glog.V(2).Infof("[%v] Pool %v: Creating new node", kluster.Name, pool.Name)

	userdata, err := templates.Ignition.GenerateNode(kluster, launchctl.Clients.Kubernetes)
	if err != nil {
		glog.Errorf("Ignition userdata couldn't be generated: %v", err)
	}

	id, err := launchctl.Clients.Openstack.CreateNode(kluster, pool, userdata)
	if err != nil {
		return err
	}

	glog.V(2).Infof("[%v] Pool %v: Created node %v.", kluster.Name, pool.Name, id)

	launchctl.requeue(kluster)
	return nil
}

func (launchctl *LaunchControl) terminateNode(kluster *v1.Kluster, id string) error {
	err := launchctl.Clients.Openstack.DeleteNode(kluster, id)
	if err != nil {
		return err
	}

	launchctl.requeue(kluster)
	return nil
}

func (launchctl *LaunchControl) updateNodePoolStatus(kluster *v1.Kluster, newInfo v1.NodePoolInfo) error {
	copy, err := launchctl.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for i, curInfo := range copy.Status.NodePools {
		if curInfo.Name == newInfo.Name {
			copy.Status.NodePools[i] = newInfo
			_, err = launchctl.Clients.Kubernikus.Kubernikus().Klusters(copy.Namespace).Update(copy)
			return err
		}
	}

	glog.Errorf("Couldn't update Nodepool %v. It's not part of kluster %v.", newInfo.Name, copy.Name)
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

	glog.Errorf("[%v] An error occured: %v", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if launchctl.queue.NumRequeues(key) < 5 {
		glog.V(6).Infof("Error while managing nodes for kluster %q: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		launchctl.queue.AddRateLimited(key)
		return
	}

	launchctl.queue.Forget(key)
	glog.V(5).Infof("[%v] Dropping out of the queue. Too many retries...", key)
}

func starting(nodes []openstack.Node) int {
	count := 0
	for _, n := range nodes {
		if n.Starting() {
			count = count + 1
		}
	}

	return count
}

func running(nodes []openstack.Node) int {
	count := 0
	for _, n := range nodes {
		if n.Running() {
			count = count + 1
		}
	}

	return count
}

func toBeTerminated(nodes []openstack.Node) int {
	toBeTerminated := 0
	for _, n := range nodes {
		if n.Stopping() {
			continue
		}

		toBeTerminated = toBeTerminated + 1
	}

	return toBeTerminated
}
