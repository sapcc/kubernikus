package controller

import (
	"fmt"
	"path"
	"reflect"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/helm/pkg/helm"

	"strings"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"google.golang.org/grpc"
)

const (
	KLUSTER_RECHECK_INTERVAL = 5 * time.Minute
)

type GroundControl struct {
	Clients
	Factories
	Config

	queue       workqueue.RateLimitingInterface
	tprInformer cache.SharedIndexInformer
}

func NewGroundController(factories Factories, clients Clients, config Config) *GroundControl {
	operator := &GroundControl{
		Clients:     clients,
		Factories:   factories,
		Config:      config,
		queue:       workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		tprInformer: factories.Kubernikus.Kubernikus().V1().Klusters().Informer(),
	}

	operator.tprInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    operator.klusterAdd,
		UpdateFunc: operator.klusterUpdate,
		DeleteFunc: operator.klusterTerminate,
	})

	return operator
}

func (op *GroundControl) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer op.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	glog.Infof("GroundControl started!")

	for i := 0; i < threadiness; i++ {
		go wait.Until(op.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(KLUSTER_RECHECK_INTERVAL)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(2).Infof("I now would do reconciliation if its was implemented. Next run in %v", KLUSTER_RECHECK_INTERVAL)
				//op.queue.Add(true)
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	<-stopCh
}

func (op *GroundControl) runWorker() {
	for op.processNextWorkItem() {
	}
}

func (op *GroundControl) processNextWorkItem() bool {
	key, quit := op.queue.Get()
	if quit {
		return false
	}
	defer op.queue.Done(key)

	err := op.handler(key.(string))
	if err == nil {
		op.queue.Forget(key)
		return true
	}

	glog.Warningf("Error running handler: %v", err)
	op.queue.AddRateLimited(key)

	return true
}

func (op *GroundControl) handler(key string) error {
	obj, exists, err := op.tprInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("Failed to fetch key %s from cache: %s", key, err)
	}
	if !exists {
		glog.Infof("TPR of kluster %s deleted", key)
	} else {
		tpr := obj.(*v1.Kluster)
		switch state := tpr.Status.State; state {
		case v1.KlusterPending:
			{
				glog.Infof("Creating Kluster %s", tpr.GetName())
				if err := op.updateStatus(tpr, v1.KlusterCreating, "Creating Cluster"); err != nil {
					glog.Errorf("Failed to update status of kluster %s:%s", tpr.GetName(), err)
				}
				if err := op.createKluster(tpr); err != nil {
					glog.Errorf("Creating kluster %s failed: %s", tpr.GetName(), err)
					if err := op.updateStatus(tpr, v1.KlusterError, err.Error()); err != nil {
						glog.Errorf("Failed to update status of kluster %s:%s", tpr.GetName(), err)
					}
					//We are making this a permanent error for now to avoid stomping the parent kluster
					return nil
				}
				glog.Infof("Kluster %s created", tpr.GetName())
			}
		case v1.KlusterTerminating:
			{
				glog.Infof("Terminating Kluster %s", tpr.GetName())
				if err := op.terminateKluster(tpr); err != nil {
					glog.Errorf("Failed to terminate kluster %s: %s", tpr.Name, err)
					return err
				}
				glog.Infof("Terminated kluster %s", tpr.GetName())
				return nil
			}
		}
	}
	return nil
}

func (op *GroundControl) klusterAdd(obj interface{}) {
	c := obj.(*v1.Kluster)
	key, err := cache.MetaNamespaceKeyFunc(c)
	if err != nil {
		return
	}
	glog.Infof("Added kluster TPR %s", key)
	op.queue.Add(key)
}

func (op *GroundControl) klusterTerminate(obj interface{}) {
	c := obj.(*v1.Kluster)
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(c)
	if err != nil {
		return
	}
	glog.Infof("Deleted kluster TPR %s", key)
	op.queue.Add(key)
}

func (op *GroundControl) klusterUpdate(cur, old interface{}) {
	curKluster := cur.(*v1.Kluster)
	oldKluster := old.(*v1.Kluster)
	if !reflect.DeepEqual(oldKluster, curKluster) {
		key, err := cache.MetaNamespaceKeyFunc(curKluster)
		if err != nil {
			return
		}
		glog.Infof("Updated kluster TPR %s", key)
		op.queue.Add(key)
	}
}

func (op *GroundControl) updateStatus(tpr *v1.Kluster, state v1.KlusterState, message string) error {
	//Get a fresh copy from the cache
	op.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().GetStore()

	obj, exists, err := op.tprInformer.GetStore().Get(tpr)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Not found cache: %#v", tpr)
	}

	kluster := obj.(*v1.Kluster)

	//Never modify the cache, at leasts thats what I've been told
	tpr, err = op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	tpr.Status.Message = message
	tpr.Status.State = state

	_, err = op.Clients.Kubernikus.Kubernikus().Klusters(tpr.Namespace).Update(tpr)
	return err
}

func (op *GroundControl) createKluster(tpr *v1.Kluster) error {

	cluster, err := ground.NewCluster(tpr.GetName(), "kluster.staging.cloud.sap")
	if err != nil {
		return err
	}

	cluster.OpenStack.AuthURL = op.Config.Openstack.AuthURL
	if err := cluster.DiscoverValues(tpr.GetName(), tpr.Account(), op.Clients.Openstack); err != nil {
		return err
	}

	//Generate helm values from cluster struct
	rawValues, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}
	glog.Infof("Installing helm release %s", tpr.GetName())
	glog.V(3).Infof("Chart values:\n%s", string(rawValues))

	_, err = op.Clients.Helm.InstallRelease(path.Join(op.Config.Helm.ChartDirectory, "kube-master"), tpr.Namespace, helm.ValueOverrides(rawValues), helm.ReleaseName(tpr.GetName()))
	return err
}

func (op *GroundControl) terminateKluster(tpr *v1.Kluster) error {
	glog.Infof("Deleting helm release %s", tpr.GetName())
	_, err := op.Clients.Helm.DeleteRelease(tpr.GetName(), helm.DeletePurge(true))
	if err != nil && !strings.Contains(grpc.ErrorDesc(err), fmt.Sprintf(`release: "%s" not found`, tpr.GetName())) {
		return err
	}
	u := serviceUsername(tpr.GetName())
	glog.Infof("Deleting openstack user %s@default", u)
	if err := op.Clients.Openstack.DeleteUser(u, "default"); err != nil {
		return err
	}

	return op.Clients.Kubernikus.Kubernikus().Klusters(tpr.Namespace).Delete(tpr.Name, &metav1.DeleteOptions{})
}

func serviceUsername(name string) string {
	return fmt.Sprintf("kubernikus-%s", name)
}
