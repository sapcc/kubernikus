package controller

import (
	"fmt"
	"path"
	"reflect"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/Masterminds/goutils"
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
		glog.V(5).Infof("Handling kluster %v in state %q", tpr.Name, tpr.Status.State)

		switch state := tpr.Status.State; state {
		case v1.KlusterPending:
			{
				if op.requiresOpenstackInfo(tpr) {
					if err := op.discoverOpenstackInfo(tpr); err != nil {
						glog.Errorf("[%v] Discovery of openstack parameters failed: %s", tpr.GetName(), err)
						if err := op.updateStatus(tpr, v1.KlusterError, err.Error()); err != nil {
							glog.Errorf("Failed to update status of kluster %s:%s", tpr.GetName(), err)
						}
						return err
					}
					return nil
				}

				if op.requiresKubernikusInfo(tpr) {
					if err := op.discoverKubernikusInfo(tpr); err != nil {
						glog.Errorf("[%v] Discovery of kubernikus parameters failed: %s", tpr.GetName(), err)
						if err := op.updateStatus(tpr, v1.KlusterError, err.Error()); err != nil {
							glog.Errorf("Failed to update status of kluster %s:%s", tpr.GetName(), err)
						}
						return err
					}
					return nil
				}

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
	cluster, err := ground.NewCluster(tpr, op.Config.Openstack.AuthURL)
	if err != nil {
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

	glog.Infof("Deleting openstack user %s@%s", tpr.Spec.OpenstackInfo.Username, tpr.Spec.OpenstackInfo.Domain)
	if err := op.Clients.Openstack.DeleteUser(tpr.Spec.OpenstackInfo.Username, tpr.Spec.OpenstackInfo.Domain); err != nil {
		return err
	}

	return op.Clients.Kubernikus.Kubernikus().Klusters(tpr.Namespace).Delete(tpr.Name, &metav1.DeleteOptions{})
}

func (op *GroundControl) requiresOpenstackInfo(kluster *v1.Kluster) bool {
	return kluster.Spec.OpenstackInfo.ProjectID == "" ||
		kluster.Spec.OpenstackInfo.RouterID == "" ||
		kluster.Spec.OpenstackInfo.NetworkID == "" ||
		kluster.Spec.OpenstackInfo.LBSubnetID == "" ||
		kluster.Spec.OpenstackInfo.Domain == "" ||
		kluster.Spec.OpenstackInfo.Region == "" ||
		kluster.Spec.OpenstackInfo.Username == "" ||
		kluster.Spec.OpenstackInfo.Password == "" ||
		kluster.Spec.OpenstackInfo.AuthURL == ""

}

func (op *GroundControl) requiresKubernikusInfo(kluster *v1.Kluster) bool {
	return kluster.Spec.KubernikusInfo.Server == "" ||
		kluster.Spec.KubernikusInfo.ServerURL == ""
}

func (op *GroundControl) discoverKubernikusInfo(kluster *v1.Kluster) error {
	glog.V(5).Infof("[%v] Discovering KubernikusInfo", kluster.Name)

	copy, err := op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if copy.Spec.KubernikusInfo.Server == "" {
		copy.Spec.KubernikusInfo.Server = fmt.Sprintf("%s.%s", kluster.GetName(), op.Config.Kubernikus.Domain)
		glog.V(5).Infof("[%v] Setting Server to %v", kluster.Name, copy.Spec.KubernikusInfo.Server)
	}

	if copy.Spec.KubernikusInfo.ServerURL == "" {
		copy.Spec.KubernikusInfo.ServerURL = fmt.Sprintf("https://%s.%s", kluster.GetName(), op.Config.Kubernikus.Domain)
		glog.V(5).Infof("[%v] Setting Server to %v", kluster.Name, copy.Spec.KubernikusInfo.ServerURL)
	}

	_, err = op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Update(copy)
	return err
}

func (op *GroundControl) discoverOpenstackInfo(kluster *v1.Kluster) error {
	glog.V(5).Infof("[%v] Discovering OpenstackInfo", kluster.Name)

	routers, err := op.Clients.Openstack.GetRouters(kluster.Account())
	if err != nil {
		return err
	}

	copy, err := op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if copy.Spec.OpenstackInfo.ProjectID == "" {
		copy.Spec.OpenstackInfo.ProjectID = kluster.Account()
		glog.V(5).Infof("[%v] Setting ProjectID to %v", kluster.Name, copy.Spec.OpenstackInfo.ProjectID)
	}

	if copy.Spec.OpenstackInfo.RouterID == "" {
		if len(routers) == 1 {
			copy.Spec.OpenstackInfo.RouterID = routers[0].ID
			glog.V(5).Infof("[%v] Setting RouterID to %v", kluster.Name, copy.Spec.OpenstackInfo.RouterID)
		} else {
			glog.V(5).Infof("[%v] There's more than 1 router. Autodiscovery not possible!")
		}
	}

	if copy.Spec.OpenstackInfo.NetworkID == "" {
		if len(routers) == 1 {
			if len(routers[0].Networks) == 1 {
				copy.Spec.OpenstackInfo.NetworkID = routers[0].Networks[0].ID
				glog.V(5).Infof("[%v] Setting NetworkID to %v", kluster.Name, copy.Spec.OpenstackInfo.NetworkID)
			} else {
				glog.V(5).Infof("[%v] There's more than 1 network on the router. Autodiscovery not possible!")
			}
		}
	}

	if copy.Spec.OpenstackInfo.LBSubnetID == "" {
		if len(routers) == 1 {
			if len(routers[0].Subnets) == 1 {
				copy.Spec.OpenstackInfo.LBSubnetID = routers[0].Subnets[0].ID
				glog.V(5).Infof("[%v] Setting LBSubnetID to %v", kluster.Name, copy.Spec.OpenstackInfo.LBSubnetID)
			} else {
				glog.V(5).Infof("[%v] There's more than 1 subnet on the router. Autodiscovery not possible!")
			}
		}
	}

	if copy.Spec.OpenstackInfo.Domain == "" {
		glog.V(5).Infof("[%v] Setting domain to %v", kluster.Name, "kubernikus")
		copy.Spec.OpenstackInfo.Domain = "kubernikus"
	}

	if copy.Spec.OpenstackInfo.Region == "" {
		copy.Spec.OpenstackInfo.Region, err = op.Clients.Openstack.GetRegion()
		if err != nil {
			return err
		}
		glog.V(5).Infof("[%v] Setting region to %v", kluster.Name, copy.Spec.OpenstackInfo.Region)
	}

	if copy.Spec.OpenstackInfo.AuthURL == "" {
		copy.Spec.OpenstackInfo.AuthURL = op.Config.Openstack.AuthURL
		glog.V(5).Infof("[%v] Setting authURL to %v", kluster.Name, op.Config.Openstack.AuthURL)
	}

	if copy.Spec.OpenstackInfo.Username == "" {
		copy.Spec.OpenstackInfo.Username = fmt.Sprintf("kubernikus-%s", kluster.Name)
		glog.V(5).Infof("[%v] Setting Username to %v", kluster.Name, copy.Spec.OpenstackInfo.Username)
	}

	if copy.Spec.OpenstackInfo.Password == "" {
		glog.V(5).Infof("[%v] Setting Password to %v", kluster.Name, "[redacted]")
		if copy.Spec.OpenstackInfo.Password, err = goutils.RandomAscii(20); err != nil {
			return fmt.Errorf("Failed to generate password: %s", err)
		}
	}

	_, err = op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Update(copy)
	return err
}
