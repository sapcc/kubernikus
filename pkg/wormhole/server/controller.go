package server

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/databus23/guttle"
	"github.com/go-kit/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	knftables "github.com/sapcc/kubernikus/pkg/util/nftables"
)

type Controller struct {
	nodes       informers.NodeInformer
	tunnel      *guttle.Server
	queue       workqueue.RateLimitingInterface // nolint: staticcheck
	store       map[string][]route
	storeMu     sync.RWMutex
	nft         knftables.Interface
	hijackPort  int
	serviceCIDR string

	Logger log.Logger
}

type route struct {
	cidr       string
	identifier string
}

func NewController(informer informers.NodeInformer, serviceCIDR string, tunnel *guttle.Server, logger log.Logger) (*Controller, error) {
	logger = log.With(logger, "controller", "tunnel")

	nft, err := knftables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize nftables: %w", err)
	}

	c := &Controller{
		nodes:       informer,
		tunnel:      tunnel,
		queue:       workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)), // nolint: staticcheck
		store:       make(map[string][]route),
		nft:         nft,
		hijackPort:  9191,
		serviceCIDR: serviceCIDR,
		Logger:      logger,
	}

	//Always forward requests to the serviceCIDR range
	c.tunnel.AddRoute(serviceCIDR)

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
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNode := oldObj.(*v1.Node)
			newNode := newObj.(*v1.Node)
			if oldNode.Spec.PodCIDR != newNode.Spec.PodCIDR {
				if key, err := cache.MetaNamespaceKeyFunc(newObj); err == nil {
					c.queue.Add(key)
				}
			}
		},
	})

	return c, nil
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer c.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	c.Logger.Log(
		"msg", "starting WormholeGenerator",
		"threadiness", threadiness)

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.recheckNodes()
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	<-stopCh
}

func (c *Controller) recheckNodes() {
	c.Logger.Log(
		"msg", "Running periodic recheck. Queuing all known nodes...",
		"v", 5)
	c.storeMu.RLock()
	defer c.storeMu.RUnlock()
	for key := range c.store {
		c.queue.Add(key)
	}
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
	c.Logger.Log(
		"msg", "requeuing because of error",
		"key", key,
		"err", err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.Logger.Log(
		"msg", "dropping because of too many error",
		"key", key)
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

	identifier := fmt.Sprintf("system:node:%v", node.GetName())

	podCIDR := node.Spec.PodCIDR
	if podCIDR == "" {
		c.Logger.Log(
			"msg", "removing tunnel routes for node with empty spec.PodCIDR",
			"node", identifier,
		)
		return c.syncRules()
	}

	c.Logger.Log(
		"msg", "adding tunnel routes",
		"node", identifier)

	ip, err := GetNodeHostIP(node)
	if err != nil {
		return err
	}
	nodeCIDR := ip.String() + "/32"

	if err := c.tunnel.AddClientRoute(podCIDR, identifier); err != nil {
		return err
	}
	c.storeRoute(key, route{cidr: podCIDR, identifier: identifier})
	if err := c.tunnel.AddRoute(podCIDR); err != nil {
		return err
	}
	if err := c.tunnel.AddClientRoute(nodeCIDR, identifier); err != nil {
		return err
	}
	c.storeRoute(key, route{cidr: nodeCIDR, identifier: identifier})
	if err := c.tunnel.AddRoute(nodeCIDR); err != nil {
		return err
	}

	return c.syncRules()
}

func (c *Controller) storeRoute(key string, r route) {
	c.storeMu.Lock()
	defer c.storeMu.Unlock()
	c.store[key] = append(c.store[key], r)
}

func (c *Controller) delNode(key string) error {
	c.storeMu.RLock()
	routes := c.store[key]
	for _, route := range routes {
		c.tunnel.DeleteClientRoute(route.cidr, route.identifier)
		c.tunnel.DeleteRoute(route.cidr)
	}
	c.storeMu.RUnlock()
	return c.syncRules()
}

func (c *Controller) syncRules() error {
	c.storeMu.RLock()
	defer c.storeMu.RUnlock()

	var cidrs []string
	for _, routes := range c.store {
		for _, r := range routes {
			cidrs = append(cidrs, r.cidr)
		}
	}
	cidrs = append(cidrs, c.serviceCIDR)

	c.Logger.Log(
		"msg", "syncing nftables rules",
		"cidrs", cidrs,
		"v", 6)

	return c.nft.SyncRules(cidrs, c.hijackPort)
}

func GetNodeHostIP(node *v1.Node) (net.IP, error) {
	addresses := node.Status.Addresses
	addressMap := make(map[v1.NodeAddressType][]v1.NodeAddress)
	for i := range addresses {
		addressMap[addresses[i].Type] = append(addressMap[addresses[i].Type], addresses[i])
	}
	if addresses, ok := addressMap[v1.NodeInternalIP]; ok {
		return net.ParseIP(addresses[0].Address), nil
	}
	if addresses, ok := addressMap[v1.NodeExternalIP]; ok {
		return net.ParseIP(addresses[0].Address), nil
	}
	return nil, fmt.Errorf("host IP unknown; known addresses: %v", addresses)
}
