package base

import (
	"errors"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	BASE_DELAY               = 5 * time.Second
	MAX_DELAY                = 300 * time.Second
	KLUSTER_RECHECK_INTERVAL = 5 * time.Minute
)

var ErrUnkownKluster = errors.New("unkown kluster")

type Controller interface {
	Run(int, <-chan struct{}, *sync.WaitGroup)
}

type Reconciler interface {
	Reconcile(kluster *v1.Kluster) (bool, error)
}

type controller struct {
	config.Factories
	config.Clients

	queue      workqueue.RateLimitingInterface
	reconciler Reconciler

	logger log.Logger
}

func NewController(factories config.Factories, clients config.Clients, reconciler Reconciler, logger log.Logger) Controller {
	c := &controller{
		Factories:  factories,
		Clients:    clients,
		queue:      workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(BASE_DELAY, MAX_DELAY)),
		reconciler: reconciler,
		logger:     logger,
	}

	c.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
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

func (c *controller) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	c.logger.Log(
		"msg", "starting run loop",
		"threadiness", threadiness,
		"v", 2,
	)

	defer c.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(KLUSTER_RECHECK_INTERVAL)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.requeueAllKlusters()
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	<-stopCh
}

func (c *controller) requeueAllKlusters() (err error) {
	defer func() {
		c.logger.Log(
			"msg", "requeued all",
			"v", 1,
			"err", err,
		)
	}()

	klusters, err := c.Factories.Kubernikus.Kubernikus().V1().Klusters().Lister().List(labels.Everything())
	if err != nil {
		return err
	}

	for _, kluster := range klusters {
		c.requeueKluster(kluster)
	}

	return nil
}

func (c *controller) requeueKluster(kluster *v1.Kluster) {
	c.logger.Log(
		"msg", "queuing",
		"kluster", kluster.Spec.Name,
		"project", kluster.Account(),
		"v", 2,
	)

	key, err := cache.MetaNamespaceKeyFunc(kluster)
	if err == nil {
		c.queue.Add(key)
	}
}

func (c *controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	var err error
	var kluster *v1.Kluster
	var requeue bool

	obj, exists, _ := c.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().GetIndexer().GetByKey(key.(string))
	if !exists {
		err = ErrUnkownKluster
	} else {
		kluster = obj.(*v1.Kluster)
	}

	if err == nil {
		// Invoke the method containing the business logic
		requeue, err = c.reconciler.Reconcile(kluster)
	}

	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.

		if requeue == false {
			c.queue.Forget(key)
		} else {
			// Requeue requested
			c.queue.AddAfter(key, BASE_DELAY)
		}

		return true
	}

	if c.queue.NumRequeues(key) < 5 {
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return true
	}

	// Retries exceeded. Forgetting for this reconciliation loop
	c.queue.Forget(key)
	return true
}
