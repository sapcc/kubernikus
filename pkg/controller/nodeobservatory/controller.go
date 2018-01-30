package nodeobservatory

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	core_v1 "k8s.io/api/core/v1"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

const (
	BaseDelay              = 5 * time.Second
	MaxDelay               = 300 * time.Second
	KlusterRecheckInterval = 5 * time.Minute
)

type (
	AddFunc    func(kluster *v1.Kluster, node *core_v1.Node)
	UpdateFunc func(kluster *v1.Kluster, nodeCur, nodeOld *core_v1.Node)
	DeleteFunc func(kluster *v1.Kluster, node *core_v1.Node)

	NodeObservatory struct {
		config.Controller
		config.Factories
		config.Clients
		namespace           string
		queue               workqueue.RateLimitingInterface
		logger              log.Logger
		nodeInformerMap     sync.Map
		handlersMux         sync.RWMutex
		addEventHandlers    []AddFunc
		updateEventHandlers []UpdateFunc
		deleteEventHandlers []DeleteFunc
		stopCh              <-chan struct{}
		informerWg          *sync.WaitGroup
		threadiness         int
	}
)

func NewController(factories config.Factories, clients config.Clients, logger log.Logger, namespace string, threadiness int) *NodeObservatory {
	logger = log.With(logger,
		"controller", "nodeobservatory",
		"threadiness", threadiness,
	)

	controller := &NodeObservatory{
		Factories:       factories,
		Clients:         clients,
		namespace:       namespace,
		queue:           workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(BaseDelay, MaxDelay)),
		logger:          logger,
		nodeInformerMap: sync.Map{},
		threadiness:     threadiness,
	}

	controller.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.klusterAddFunc,
		UpdateFunc: controller.klusterUpdateFunc,
		DeleteFunc: controller.klusterDeleteFunc,
	})

	return controller
}

func (n *NodeObservatory) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	n.logger.Log(
		"msg", "starting run loop",
		"v", 2,
	)

	n.stopCh = stopCh
	n.informerWg = wg

	defer n.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)

	for i := 0; i < n.threadiness; i++ {
		go wait.Until(n.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(KlusterRecheckInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				n.requeueAllKlusters()
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()
	<-stopCh
}

func (n *NodeObservatory) requeueAllKlusters() (err error) {
	defer func() {
		n.logger.Log(
			"msg", "requeued all",
			"v", 1,
			"err", err,
		)
	}()

	klusters, err := n.Factories.Kubernikus.Kubernikus().V1().Klusters().Lister().Klusters(n.namespace).List(labels.Everything())
	if err != nil {
		return err
	}

	for _, kluster := range klusters {
		n.requeueKluster(kluster)
	}

	return nil
}

func (n *NodeObservatory) requeueKluster(kluster *v1.Kluster) {
	n.logger.Log(
		"msg", "queuing",
		"kluster", kluster.Spec.Name,
		"project", kluster.Account(),
		"v", 2,
	)

	key, err := cache.MetaNamespaceKeyFunc(kluster)
	if err == nil {
		n.queue.Add(key)
	}
}

func (n *NodeObservatory) runWorker() {
	for n.processNextWorkItem() {
	}
}

func (n *NodeObservatory) processNextWorkItem() bool {
	key, quit := n.queue.Get()
	if quit {
		return false
	}
	defer n.queue.Done(key)

	var kluster *v1.Kluster
	var requeue bool

	if obj, exists, _ := n.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().GetIndexer().GetByKey(key.(string)); exists {
		kluster = obj.(*v1.Kluster)
	}

	// Invoke the method containing the business logic
	requeue, err := n.reconcile(kluster)

	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.

		if requeue == false {
			n.queue.Forget(key)
		} else {
			// Requeue requested
			n.queue.AddAfter(key, BaseDelay)
		}

		return true
	}

	if n.queue.NumRequeues(key) < 5 {
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		n.queue.AddRateLimited(key)
		return true
	}

	// Retries exceeded. Forgetting for this reconciliation loop
	n.queue.Forget(key)
	return true
}

func (n *NodeObservatory) reconcile(kluster *v1.Kluster) (bool, error) {

	n.cleanUpInformers()

	if kluster != nil {
		if err := n.createAndWatchNodeInformerForKluster(kluster); err != nil {
			return true, err
		}
	}

	return false, nil
}

func (n *NodeObservatory) cleanUpInformers() {
	n.nodeInformerMap.Range(
		func(key, value interface{}) bool {
			if _, exists, _ := n.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().GetIndexer().GetByKey(key.(string)); !exists {
				if i, ok := n.nodeInformerMap.Load(key); ok {
					informer := i.(*NodeInformer)
					informer.close()
				}
				n.nodeInformerMap.Delete(key)
			}
			return true
		},
	)
}

func (n *NodeObservatory) createAndWatchNodeInformerForKluster(kluster *v1.Kluster) error {
	key, err := cache.MetaNamespaceKeyFunc(kluster)
	if err != nil {
		return err
	}

	if _, exists := n.nodeInformerMap.Load(key); !exists {
		n.logger.Log(
			"creating nodeInformer for kluster %s", key,
			"v", 2,
		)
		nodeInformer, err := newNodeInformerForKluster(n.Clients.Satellites, kluster)
		if err != nil {
			return err
		}

		nodeInformer.SharedIndexInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if kluster, err = n.getKlusterByKey(key); err != nil {
					n.logger.Log(err)
					return
				}
				n.handlersMux.RLock()
				defer n.handlersMux.RUnlock()
				for _, addHandler := range n.addEventHandlers {
					addHandler(kluster, obj.(*core_v1.Node))
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if kluster, err = n.getKlusterByKey(key); err != nil {
					n.logger.Log(
						"err", err,
						"v", 2,
					)
					return
				}
				n.handlersMux.RLock()
				defer n.handlersMux.RUnlock()
				for _, updateHandler := range n.updateEventHandlers {
					updateHandler(kluster, oldObj.(*core_v1.Node), newObj.(*core_v1.Node))
				}
			},
			DeleteFunc: func(obj interface{}) {
				if kluster, err = n.getKlusterByKey(key); err != nil {
					n.logger.Log(
						"err", err,
						"v", 2,
					)
					return
				}
				n.handlersMux.RLock()
				defer n.handlersMux.RUnlock()
				for _, deleteHandler := range n.deleteEventHandlers {
					deleteHandler(kluster, obj.(*core_v1.Node))
				}
			},
		})

		n.nodeInformerMap.Store(
			key,
			nodeInformer,
		)

		go func(informer *NodeInformer) {
			n.informerWg.Add(1)
			defer n.informerWg.Done()
			ch := make(chan struct{})

			go func() {
				informer.run()
				close(ch)
			}()

			select {
			case <-ch:
			case <-n.stopCh:
				informer.close()
			}

		}(nodeInformer)

	}
	return nil
}

func (n *NodeObservatory) getKlusterByKey(key string) (*v1.Kluster, error) {
	o, exists, err := n.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("kluster %v was not found", key)
	}
	return o.(*v1.Kluster), err
}

func (n *NodeObservatory) klusterAddFunc(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		n.queue.Add(key)
	}
}

func (n *NodeObservatory) klusterUpdateFunc(cur, old interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(cur)
	if err == nil {
		n.queue.Add(key)
	}
}

func (n *NodeObservatory) klusterDeleteFunc(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		n.queue.Add(key)
	}
}

// GetStoreForKluster returns the SharedIndexInformers cache.Store for a given kluster or an error
func (n *NodeObservatory) GetStoreForKluster(kluster *v1.Kluster) (cache.Store, error) {
	informer, err := n.getNodeInformerForKluster(kluster)
	if err != nil {
		return nil, err
	}
	return informer.GetStore(), nil
}

// GetIndexerForKluster returns the SharedIndexInformers cache.Indexer for a given kluster or an error
func (n *NodeObservatory) GetIndexerForKluster(kluster *v1.Kluster) (cache.Indexer, error) {
	informer, err := n.getNodeInformerForKluster(kluster)
	if err != nil {
		return nil, err
	}
	return informer.GetIndexer(), nil
}

func (n *NodeObservatory) getNodeInformerForKluster(kluster *v1.Kluster) (*NodeInformer, error) {
	key, err := cache.MetaNamespaceKeyFunc(kluster)
	if err != nil {
		return nil, err
	}
	informer, ok := n.nodeInformerMap.Load(key)
	if ok {
		return informer.(*NodeInformer), nil
	}
	return nil, fmt.Errorf("no informer found for kluster %v", key)
}

// HasSyncedForKluster returns true if the store of the kluster's SharedIndexInformer has synced.
func (n *NodeObservatory) HasSyncedForKluster(kluster *v1.Kluster) bool {
	key, err := cache.MetaNamespaceKeyFunc(kluster)
	if err != nil {
		return false
	}
	informer, ok := n.nodeInformerMap.Load(key)
	if ok {
		return false
	}
	return informer.(*NodeInformer).HasSynced()
}

// AddEventHandlerFuncs adds event handlers to the SharedIndexInformer
func (n *NodeObservatory) AddEventHandlerFuncs(addFunc AddFunc, updateFunc UpdateFunc, deleteFunc DeleteFunc) {
	n.handlersMux.Lock()
	defer n.handlersMux.Unlock()

	if addFunc != nil {
		n.addEventHandlers = append(n.addEventHandlers, addFunc)
	}
	if updateFunc != nil {
		n.updateEventHandlers = append(n.updateEventHandlers, updateFunc)
	}
	if deleteFunc != nil {
		n.deleteEventHandlers = append(n.deleteEventHandlers, deleteFunc)
	}
}
