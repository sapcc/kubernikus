package ground

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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/helm/pkg/helm"

	"strings"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	helmutil "github.com/sapcc/kubernikus/pkg/client/helm"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	"github.com/sapcc/kubernikus/pkg/version"
	"google.golang.org/grpc"
)

const (
	TPR_RECHECK_INTERVAL = 5 * time.Minute
	CACHE_RESYNC_PERIOD  = 2 * time.Minute
)

type Options struct {
	KubeConfig string

	ChartDirectory    string
	AuthURL           string
	AuthUsername      string
	AuthPassword      string
	AuthDomain        string
	AuthProject       string
	AuthProjectDomain string
}

type Operator struct {
	Options

	kubernikusClient clientset.Interface
	kubernetesClient kubernetes.Interface
	kubernetesConfig *rest.Config
	openstackClient  openstack.Client

	informers   kubernikus_informers.SharedInformerFactory
	tprInformer cache.SharedIndexInformer
	queue       workqueue.RateLimitingInterface
}

func New(options Options) *Operator {
	kubernetesClient, err := kube.NewKubernetesClient(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes clients: %s", err)
	}

	kubernetesConfig, err := kube.Config(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes config: %s", err)
	}

	kubernikusClient, err := kubernikus.NewKubernikusClient(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernikus clients: %s", err)
	}

	openstackClient, err := openstack.NewClient(
		options.AuthURL,
		options.AuthUsername,
		options.AuthPassword,
		options.AuthDomain,
		options.AuthProject,
		options.AuthProjectDomain,
	)
	if err != nil {
		glog.Fatalf("Failed to create openstack client: %s", err)
	}

	operator := &Operator{
		Options:          options,
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		kubernikusClient: kubernikusClient,
		kubernetesClient: kubernetesClient,
		openstackClient:  openstackClient,
		kubernetesConfig: kubernetesConfig,
	}

	operator.informers = kubernikus_informers.NewSharedInformerFactory(operator.kubernikusClient, 5*time.Minute)

	operator.informers.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    operator.klusterAdd,
		UpdateFunc: operator.klusterUpdate,
		DeleteFunc: operator.klusterTerminate,
	})

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

	err := op.handler(key.(string))
	if err == nil {
		op.queue.Forget(key)
		return true
	}

	glog.Warningf("Error running handler: %v", err)
	op.queue.AddRateLimited(key)

	return true
}

func (op *Operator) handler(key string) error {
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

func (op *Operator) klusterAdd(obj interface{}) {
	c := obj.(*v1.Kluster)
	key, err := cache.MetaNamespaceKeyFunc(c)
	if err != nil {
		return
	}
	glog.Infof("Added kluster TPR %s", key)
	op.queue.Add(key)
}

func (op *Operator) klusterTerminate(obj interface{}) {
	c := obj.(*v1.Kluster)
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(c)
	if err != nil {
		return
	}
	glog.Infof("Deleted kluster TPR %s", key)
	op.queue.Add(key)
}

func (op *Operator) klusterUpdate(cur, old interface{}) {
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

func (op *Operator) updateStatus(tpr *v1.Kluster, state v1.KlusterState, message string) error {
	//Get a fresh copy from the cache
	obj, exists, err := op.tprInformer.GetStore().Get(tpr)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Not found cache: %#v", tpr)
	}

	kluster := obj.(*v1.Kluster)

	//Never modify the cache, at leasts thats what I've been told
	tpr, err = op.kubernikusClient.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	tpr.Status.Message = message
	tpr.Status.State = state

	_, err = op.kubernikusClient.Kubernikus().Klusters(tpr.Namespace).Update(tpr)
	return err
}

func (op *Operator) createKluster(tpr *v1.Kluster) error {
	helmClient, err := helmutil.NewClient(op.kubernetesClient, op.kubernetesConfig)
	if err != nil {
		return fmt.Errorf("Failed to create helm client: %s", err)
	}

	routers, err := op.openstackClient.GetRouters(tpr.Account())
	if err != nil {
		return fmt.Errorf("Couldn't get routers for project %s: %s", tpr.Account(), err)
	}

	glog.V(2).Infof("Found routers for project %s: %#v", tpr.Account(), routers)

	if !(len(routers) == 1 && len(routers[0].Subnets) == 1) {
		return fmt.Errorf("Project needs to contain a router with exactly one subnet")
	}

	cluster, err := NewCluster(tpr.GetName(), "kluster.staging.cloud.sap")
	if err != nil {
		return err
	}

	cluster.OpenStack.AuthURL = op.AuthURL
	cluster.OpenStack.Username = serviceUsername(tpr.GetName())
	password, err := goutils.RandomAscii(20)
	if err != nil {
		return fmt.Errorf("Failed to generate password: %s", err)
	}
	cluster.OpenStack.Password = password
	cluster.OpenStack.DomainName = "Default"
	cluster.OpenStack.ProjectID = tpr.Account()
	cluster.OpenStack.RouterID = routers[0].ID
	cluster.OpenStack.LBSubnetID = routers[0].Subnets[0].ID

	//Generate helm values from cluster struct
	rawValues, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}
	glog.Infof("Installing helm release %s", tpr.GetName())
	glog.V(3).Infof("Chart values:\n%s", string(rawValues))

	_, err = helmClient.InstallRelease(path.Join(op.ChartDirectory, "kube-master"), tpr.Namespace, helm.ValueOverrides(rawValues), helm.ReleaseName(tpr.GetName()))
	return err
}

func (op *Operator) terminateKluster(tpr *v1.Kluster) error {
	helmClient, err := helmutil.NewClient(op.kubernetesClient, op.kubernetesConfig)
	if err != nil {
		return fmt.Errorf("Failed to create helm client: %s", err)
	}
	glog.Infof("Deleting helm release %s", tpr.GetName())
	_, err = helmClient.DeleteRelease(tpr.GetName(), helm.DeletePurge(true))
	if err != nil && !strings.Contains(grpc.ErrorDesc(err), fmt.Sprintf(`release: "%s" not found`, tpr.GetName())) {
		return err
	}
	u := serviceUsername(tpr.GetName())
	glog.Infof("Deleting openstack user %s@default", u)
	if err := op.openstackClient.DeleteUser(u, "default"); err != nil {
		return err
	}

	return op.kubernikusClient.Kubernikus().Klusters(tpr.Namespace).Delete(tpr.Name, &metav1.DeleteOptions{})
}

func serviceUsername(name string) string {
	return fmt.Sprintf("kubernikus-%s", name)
}
