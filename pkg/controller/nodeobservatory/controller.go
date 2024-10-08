package nodeobservatory

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	kubernikus_informers_v1 "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
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

	NodeEventHandlerFuncs struct {
		AddFunc    AddFunc
		UpdateFunc UpdateFunc
		DeleteFunc DeleteFunc
	}

	NodeObservatory struct {
		clientFactory       kube.SharedClientFactory
		klusterInformer     kubernikus_informers_v1.KlusterInformer
		namespace           string
		queue               workqueue.RateLimitingInterface // nolint: staticcheck
		logger              log.Logger
		nodeInformerMap     sync.Map
		handlersMux         sync.RWMutex
		addEventHandlers    []AddFunc
		updateEventHandlers []UpdateFunc
		deleteEventHandlers []DeleteFunc
		stopCh              <-chan struct{}
		threadiness         int
	}
)

func NewController(informer kubernikus_informers_v1.KlusterInformer, factory kube.SharedClientFactory, logger log.Logger, threadiness int) *NodeObservatory {
	logger = log.With(logger,
		"controller", "nodeobservatory",
		"threadiness", threadiness,
	)

	controller := &NodeObservatory{
		clientFactory:   factory,
		klusterInformer: informer,
		queue:           workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(BaseDelay, MaxDelay), "nodeobservatory"), // nolint: staticcheck
		logger:          logger,
		nodeInformerMap: sync.Map{},
		threadiness:     threadiness,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.klusterAddFunc,
		UpdateFunc: controller.klusterUpdateFunc,
		DeleteFunc: controller.klusterDeleteFunc,
	})

	return controller
}

func (n *NodeObservatory) Run(stopCh <-chan struct{}) {
	n.logger.Log(
		"msg", "starting run loop",
		"v", 2,
	)

	n.stopCh = stopCh

	defer n.queue.ShutDown()

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

	klusters, err := n.klusterInformer.Lister().Klusters(n.namespace).List(labels.Everything())
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

	if obj, exists, _ := n.klusterInformer.Informer().GetIndexer().GetByKey(key.(string)); exists {
		kluster = obj.(*v1.Kluster)
	}

	// Invoke the method containing the business logic
	err := n.reconcile(kluster)

	if err == nil {
		n.queue.Forget(key)
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

func (n *NodeObservatory) reconcile(kluster *v1.Kluster) error {

	n.cleanUpInformers()

	if kluster != nil && (kluster.Status.Phase == models.KlusterPhaseRunning || kluster.Status.Phase == models.KlusterPhaseUpgrading || kluster.Status.Phase == models.KlusterPhaseTerminating) {
		if err := n.createAndWatchNodeInformerForKluster(kluster); err != nil {
			return err
		}
	}

	return nil
}

func (n *NodeObservatory) cleanUpInformers() {
	n.nodeInformerMap.Range(
		func(key, value interface{}) bool {
			if _, exists, _ := n.klusterInformer.Informer().GetIndexer().GetByKey(key.(string)); !exists {
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
			"msg", "creating nodeInformer",
			"kluster", key,
			"v", 2,
		)
		nodeInformer, err := newNodeInformerForKluster(n.clientFactory, kluster)
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
					node, ok := obj.(*core_v1.Node)
					if !ok {
						tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
						if !ok {
							n.logger.Log("obj", fmt.Sprintf("%v", obj), "err", "unexpected object type")
							return
						}
						if node, ok = tombstone.Obj.(*core_v1.Node); !ok {
							n.logger.Log("obj", fmt.Sprintf("%v", tombstone.Obj), "err", "unexpected object type in tombstone.Obj")
							return
						}
					}

					deleteHandler(kluster, node)
				}
			},
		})

		n.nodeInformerMap.Store(
			key,
			nodeInformer,
		)

		go func(informer *NodeInformer) {
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
	o, exists, err := n.klusterInformer.Informer().GetIndexer().GetByKey(key)
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
func (n *NodeObservatory) GetListerForKluster(kluster *v1.Kluster) (listers_core_v1.NodeLister, error) {
	informer, err := n.getNodeInformerForKluster(kluster)
	if err != nil {
		return nil, err
	}

	if err := wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (bool, error) { return informer.HasSynced(), nil }); err != nil { //nolint:staticcheck
		return nil, errors.New("Node cache not synced")
	}
	return listers_core_v1.NewNodeLister(informer.GetIndexer()), nil
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
func (n *NodeObservatory) AddEventHandlerFuncs(handlers NodeEventHandlerFuncs) {
	n.handlersMux.Lock()
	defer n.handlersMux.Unlock()

	if handlers.AddFunc != nil {
		n.addEventHandlers = append(n.addEventHandlers, handlers.AddFunc)
	}
	if handlers.UpdateFunc != nil {
		n.updateEventHandlers = append(n.updateEventHandlers, handlers.UpdateFunc)
	}
	if handlers.DeleteFunc != nil {
		n.deleteEventHandlers = append(n.deleteEventHandlers, handlers.DeleteFunc)
	}
}
