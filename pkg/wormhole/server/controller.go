package server

import (
	"net"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/koding/tunnel"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	nodes  informers.NodeInformer
	tunnel *tunnel.Server
	queue  workqueue.RateLimitingInterface
	store  map[string]net.Listener
}

func NewController(informer informers.NodeInformer, tunnel *tunnel.Server) *Controller {
	c := &Controller{
		nodes:  informer,
		tunnel: tunnel,
		queue:  workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		store:  make(map[string]net.Listener),
	}

	c.nodes.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
	})

	return c
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer c.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	glog.Infof(`Starting WormholeGenerator with %d workers`, threadiness)

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(5).Infof("Running periodic recheck. Queuing all known nodes...")
				for key, _ := range c.store {
					c.queue.Add(key)
				}
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	<-stopCh
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.reconcile(key.(string))
	c.handleErr(err, key)
	return true
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}
	glog.Errorf("Requeuing %v: %v", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	glog.Infof("Dropping %v. Too many errors", key)
	c.queue.Forget(key)
}

func (c *Controller) reconcile(key string) error {
	obj, exists, err := c.nodes.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		return c.delNode(key)
	}

	return c.addNode(key, obj.(*v1.Node))
}

func (c *Controller) addNode(key string, node *v1.Node) error {
	if c.store[key] == nil {

		listener, err := net.Listen("tcp", "127.0.0.1:")
		if err != nil {
			return err
		}

		glog.Infof("Listening to node %v on %v", key, listener.Addr())

		c.store[key] = listener
		c.tunnel.AddAddr(listener, nil, node.Spec.ExternalID)
	} else {
		glog.V(5).Infof("Already listening on this node... Skipping %v", key)
	}
	return nil
}

func (c *Controller) delNode(key string) error {
	listener := c.store[key]
	if listener != nil {
		glog.Infof("Deleting node %v", key)
		c.tunnel.DeleteAddr(listener, nil)
		listener.Close()
		c.store[key] = nil
	} else {
		glog.V(5).Infof("Not listening on this node... Skipping %v", key)
	}
	return nil
}
