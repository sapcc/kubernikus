package ground

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/sapcc/kubernikus/pkg/kube"
	tprv1 "github.com/sapcc/kubernikus/pkg/tpr/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

const (
	TPR_RECHECK_INTERVAL = 5 * time.Minute
	CACHE_RESYNC_PERIOD  = 2 * time.Minute
)

type Options struct {
	kube.Options
}

type Operator struct {
	Options

	clientset   *kubernetes.Clientset
	tprClient   *rest.RESTClient
	tprScheme   *runtime.Scheme
	tprInformer cache.SharedIndexInformer
	queue       workqueue.RateLimitingInterface
}

func New(options Options) *Operator {

	operator := &Operator{
		Options: options,
		queue:   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	operator.clientset, operator.tprClient, operator.tprScheme = kube.NewClients(options.Options)

	tprInformer := cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(operator.tprClient, tprv1.KlusterResourcePlural, metav1.NamespaceAll, fields.Everything()),
		&tprv1.Kluster{},
		CACHE_RESYNC_PERIOD,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	tprInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    operator.klusterAdd,
		UpdateFunc: operator.klusterUpdate,
		DeleteFunc: operator.klusterDelete,
	})
	operator.tprInformer = tprInformer

	return operator
}

func (op *Operator) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer op.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	glog.Infof("Kluster operator started!  %v\n", version.VERSION)

	go op.tprInformer.Run(stopCh)

	glog.Info("Waiting for cache to sync...")
	cache.WaitForCacheSync(stopCh, op.tprInformer.HasSynced)
	glog.Info("Cache primed. Ready for operations.")

	for i := 0; i < threadiness; i++ {
		go wait.Until(op.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(TPR_RECHECK_INTERVAL)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(2).Infof("I now would do reconciliation if its was implemented. Next run in %v", TPR_RECHECK_INTERVAL)
				//op.queue.Add(true)
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	<-stopCh
}

func (op *Operator) runWorker() {
	for op.processNextWorkItem() {
	}
}

func (op *Operator) processNextWorkItem() bool {
	key, quit := op.queue.Get()
	if quit {
		return false
	}
	defer op.queue.Done(key)

	project, ok := key.(*tprv1.Kluster)
	if !ok {
		glog.Warningf("Skipping work item of unexpected type: %v", key)
		op.queue.Forget(key)
		return true
	}
	err := op.handler(project)
	if err == nil {
		op.queue.Forget(key)
		return true
	}

	glog.Warningf("Error running handler: %v", err)
	op.queue.AddRateLimited(key)

	return true
}

func (op *Operator) handler(key *tprv1.Kluster) error {
	obj, exists, err := op.tprInformer.GetStore().Get(key)
	if err != nil {
		return fmt.Errorf("Failed to fetch key %s from cache: %s", key.Name, err)
	}
	if !exists {
		glog.Infof("Deleting Cluster %s (not really, maybe in the future)", key.GetName())
	} else {
		tpr := obj.(*tprv1.Kluster)
		if tpr.Status.State == tprv1.KlusterPending {
			op.updateStatus(tpr, tprv1.KlusterCreating, "Creating Cluster")
		}
	}
	return nil
}

func (op *Operator) klusterAdd(obj interface{}) {
	c := obj.(*tprv1.Kluster)
	glog.Infof("Added cluster %s in namespace %s", c.GetName(), c.GetNamespace())
	op.queue.Add(c)
}

func (op *Operator) klusterDelete(obj interface{}) {
	c := obj.(*tprv1.Kluster)
	glog.Infof("Deleted cluster %s in namespace %s", c.GetName(), c.GetNamespace())
	op.queue.Add(c)
}

func (op *Operator) klusterUpdate(cur, old interface{}) {
	curKluster := cur.(*tprv1.Kluster)
	oldKluster := old.(*tprv1.Kluster)
	if !reflect.DeepEqual(oldKluster.Spec, curKluster.Spec) {
		glog.Infof("Updated cluster %s in namespace %s", oldKluster.GetName(), oldKluster.GetNamespace())
		op.queue.Add(cur)
	}
}

func (op *Operator) updateStatus(tpr *tprv1.Kluster, state tprv1.KlusterState, message string) error {
	r, err := op.tprScheme.Copy(tpr)
	if err != nil {
		return err
	}
	tpr = r.(*tprv1.Kluster)
	tpr.Status.Message = message
	tpr.Status.State = state

	return op.tprClient.Put().
		Name(tpr.ObjectMeta.Name).
		Namespace(tpr.ObjectMeta.Namespace).
		Resource(tprv1.KlusterResourcePlural).
		Body(tpr).
		Do().
		Error()
}
