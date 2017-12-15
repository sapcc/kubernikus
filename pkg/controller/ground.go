package controller

import (
	"fmt"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/goutils"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/helm/pkg/helm"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/util"
	helm_util "github.com/sapcc/kubernikus/pkg/util/helm"
	waitutil "github.com/sapcc/kubernikus/pkg/util/wait"
)

const (
	KLUSTER_RECHECK_INTERVAL = 5 * time.Minute

	//Reason constants for the event recorder
	ConfigurationError = "ConfigurationError"
	FailedCreate       = "FailedCreate"
)

type GroundControl struct {
	config.Clients
	config.Factories
	config.Config
	Recorder record.EventRecorder

	queue           workqueue.RateLimitingInterface
	klusterInformer cache.SharedIndexInformer
	podInformer     cache.SharedIndexInformer
}

func NewGroundController(factories config.Factories, clients config.Clients, recorder record.EventRecorder, config config.Config) *GroundControl {
	operator := &GroundControl{
		Clients:         clients,
		Factories:       factories,
		Config:          config,
		Recorder:        recorder,
		queue:           workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		klusterInformer: factories.Kubernikus.Kubernikus().V1().Klusters().Informer(),
		podInformer:     factories.Kubernetes.Core().V1().Pods().Informer(),
	}

	operator.klusterInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    operator.klusterAdd,
		UpdateFunc: operator.klusterUpdate,
		DeleteFunc: operator.klusterTerminate,
	})

	operator.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    operator.podAdd,
		UpdateFunc: operator.podUpdate,
		DeleteFunc: operator.podDelete,
	})

	return operator
}

func (op *GroundControl) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer op.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	glog.Infof(`Starting GroundControl with %d "threads"`, threadiness)

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
	obj, exists, err := op.klusterInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("Failed to fetch key %s from cache: %s", key, err)
	}
	if !exists {
		glog.Infof("kluster resource %s deleted", key)
	} else {
		kluster := obj.(*v1.Kluster)
		glog.V(5).Infof("Handling kluster %v in phase %q", kluster.Name, kluster.Status.Phase)
		metrics.SetMetricKlusterInfo(kluster.GetNamespace(), kluster.GetName(), kluster.Status.Version, kluster.Spec.Openstack.ProjectID, kluster.GetAnnotations(), kluster.GetLabels())
		metrics.SetMetricKlusterStatusPhase(kluster.GetName(), kluster.Status.Phase)

		//TODO: remove ASAP, this is just a poor mans migration, adding the sec group id to existing klusters
		if err := op.ensureSecurityGroupID(kluster); err != nil {
			op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, ConfigurationError, "Failed to add default security grop id to kluster: %s", err)
		}

		switch phase := kluster.Status.Phase; phase {
		case models.KlusterPhasePending:
			{
				if op.requiresOpenstackInfo(kluster) {
					if err := op.discoverOpenstackInfo(kluster); err != nil {
						op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, ConfigurationError, "Discovery of openstack parameters failed: %s", err)
						return err
					}
					return nil
				}

				if op.requiresKubernikusInfo(kluster) {
					if err := op.discoverKubernikusInfo(kluster); err != nil {
						op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, ConfigurationError, "Discovery of kubernikus parameters failed: %s", err)
						return err
					}
					return nil
				}

				glog.Infof("Creating Kluster %s", kluster.GetName())
				if err := op.createKluster(kluster); err != nil {
					op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, FailedCreate, "Failed to create cluster: %s", err)
					return err
				}
				if err := op.updatePhase(kluster, models.KlusterPhaseCreating, "Creating Cluster"); err != nil {
					glog.Errorf("Failed to update status of kluster %s:%s", kluster.GetName(), err)
				}
				glog.Infof("Kluster %s created", kluster.GetName())
			}
		case models.KlusterPhaseCreating:
			pods, err := op.podInformer.GetIndexer().ByIndex("kluster", kluster.GetName())
			if err != nil {
				return err
			}
			podsReady := 0
			for _, obj := range pods {
				if kubernetes.IsPodReady(obj.(*api_v1.Pod)) {
					podsReady++
				}
			}
			glog.V(5).Infof("%d of %d pods ready for kluster %s", podsReady, len(pods), key)
			if podsReady == 4 {
				clientset, err := op.Clients.Satellites.ClientFor(kluster)
				if err != nil {
					return err
				}
				if err := ground.SeedKluster(clientset, kluster); err != nil {
					return err
				}
				if err := op.updatePhase(kluster, models.KlusterPhaseRunning, ""); err != nil {
					glog.Errorf("Failed to update status of kluster %s:%s", kluster.GetName(), err)
				}
				metrics.SetMetricBootDurationSummary(kluster.GetCreationTimestamp().Time, time.Now())
				glog.Infof("Kluster %s is ready!", kluster.GetName())
			}
		case models.KlusterPhaseTerminating:
			{
				glog.Infof("Terminating Kluster %s", kluster.GetName())
				if err := op.terminateKluster(kluster); err != nil {
					op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, "", "Failed to terminate cluster: %s", err)
					glog.Errorf("Failed to terminate kluster %s: %s", kluster.Name, err)
					return err
				}
				metrics.SetMetricKlusterTerminated(kluster.GetName())
				glog.Infof("Terminated kluster %s", kluster.GetName())
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
	glog.Infof("Added kluster resource %s", key)
	op.queue.Add(key)
}

func (op *GroundControl) klusterTerminate(obj interface{}) {
	c := obj.(*v1.Kluster)
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(c)
	if err != nil {
		return
	}
	glog.Infof("Deleted kluster resource %s", key)
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
		glog.Infof("Updated kluster resource %s", key)
		op.queue.Add(key)
	}
}

func (op *GroundControl) updatePhase(kluster *v1.Kluster, phase models.KlusterPhase, message string) error {

	//Never modify the cache, at leasts thats what I've been told
	kluster, err := op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	//Do nothing is the phase is not changing
	if kluster.Status.Phase == phase {
		return nil
	}
	op.Recorder.Eventf(kluster, api_v1.EventTypeNormal, string(phase), "%s cluster", phase)
	kluster.Status.Message = message
	kluster.Status.Phase = phase

	_, err = op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Update(kluster)
	if err == nil {
		//Wait for up to 5 seconds for the local cache to reflect the phase change
		waitutil.WaitForKluster(kluster, op.klusterInformer.GetIndexer(), func(k *v1.Kluster) (bool, error) {
			return k.Status.Phase == phase, nil
		})
	}
	return err
}

func (op *GroundControl) createKluster(kluster *v1.Kluster) error {
	accessMode, err := kubernetes.PVAccessMode(op.Clients.Kubernetes)
	if err != nil {
		return fmt.Errorf("Couldn't determine access mode for pvc: %s", err)
	}

	apiURL, err := op.Clients.Openstack.GetKubernikusCatalogEntry()
	if err != nil {
		return fmt.Errorf("Couldn't determine kubernikus api from service catalog: %s", err)
	}

	certificates, err := util.CreateCertificates(kluster, apiURL, op.Config.Openstack.AuthURL, op.Config.Kubernikus.Domain)
	if err != nil {
		return fmt.Errorf("Failed to generate certificates: %s", err)
	}
	bootstrapToken := util.GenerateBootstrapToken()
	username := fmt.Sprintf("kubernikus-%s", kluster.Name)
	password, err := goutils.Random(20, 32, 127, true, true)
	if err != nil {
		return fmt.Errorf("Failed to generate password: %s", err)
	}
	domain := "kubernikus"
	region, err := op.Clients.Openstack.GetRegion()
	if err != nil {
		return err
	}

	glog.Infof("Creating service user %s", username)
	if err := op.Clients.Openstack.CreateKlusterServiceUser(
		username,
		password,
		domain,
		kluster.Spec.Openstack.ProjectID,
	); err != nil {
		return err
	}

	options := &helm_util.OpenstackOptions{
		AuthURL:    op.Config.Openstack.AuthURL,
		Username:   username,
		Password:   password,
		DomainName: domain,
		Region:     region,
	}

	rawValues, err := helm_util.KlusterToHelmValues(kluster, options, certificates, bootstrapToken, accessMode)
	if err != nil {
		return err
	}
	glog.Infof("Installing helm release %s", kluster.GetName())
	glog.V(6).Infof("Chart values:\n%s", string(rawValues))

	_, err = op.Clients.Helm.InstallRelease(path.Join(op.Config.Helm.ChartDirectory, "kube-master"), kluster.Namespace, helm.ValueOverrides(rawValues), helm.ReleaseName(kluster.GetName()))
	return err
}

func (op *GroundControl) terminateKluster(kluster *v1.Kluster) error {
	if secret, err := op.Clients.Kubernetes.CoreV1().Secrets(kluster.Namespace).Get(kluster.GetName(), metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		if err != nil {
			return err
		}
		username := string(secret.Data["openstack-username"])
		domain := string(secret.Data["openstack-domain-name"])

		glog.Infof("Deleting openstack user %s@%s", username, domain)
		if err := op.Clients.Openstack.DeleteUser(username, domain); err != nil {
			return err
		}
	}

	glog.Infof("Deleting helm release %s", kluster.GetName())
	_, err := op.Clients.Helm.DeleteRelease(kluster.GetName(), helm.DeletePurge(true))
	if err != nil && !strings.Contains(grpc.ErrorDesc(err), fmt.Sprintf(`release: "%s" not found`, kluster.GetName())) {
		return err
	}

	err = op.Clients.Kubernikus.Discovery().RESTClient().Delete().AbsPath("apis/kubernikus.sap.cc/v1").
		Namespace(kluster.Namespace).
		Resource("klusters").
		Name(kluster.Name).
		Do().
		Error()

	if err == nil {
		waitutil.WaitForKlusterDeletion(kluster, op.klusterInformer.GetIndexer())
	}
	return err
}

func (op *GroundControl) requiresOpenstackInfo(kluster *v1.Kluster) bool {
	return kluster.Spec.Openstack.ProjectID == "" ||
		kluster.Spec.Openstack.RouterID == "" ||
		kluster.Spec.Openstack.NetworkID == "" ||
		kluster.Spec.Openstack.LBSubnetID == ""
}

func (op *GroundControl) requiresKubernikusInfo(kluster *v1.Kluster) bool {
	return kluster.Status.Apiserver == "" || kluster.Status.Wormhole == ""
}

func (op *GroundControl) discoverKubernikusInfo(kluster *v1.Kluster) error {
	glog.V(5).Infof("[%v] Discovering KubernikusInfo", kluster.Name)

	copy, err := op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if copy.Status.Apiserver == "" {
		copy.Status.Apiserver = fmt.Sprintf("https://%s.%s", kluster.GetName(), op.Config.Kubernikus.Domain)
		glog.Infof("[%v] Setting ServerURL to %v", kluster.Name, copy.Status.Apiserver)
	}

	if copy.Status.Wormhole == "" {
		copy.Status.Wormhole = fmt.Sprintf("https://%s-wormhole.%s", kluster.GetName(), op.Config.Kubernikus.Domain)
		glog.Infof("[%v] Setting WormholeURL to %v", kluster.Name, copy.Status.Wormhole)
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

	if copy.Spec.Openstack.ProjectID == "" {
		copy.Spec.Openstack.ProjectID = kluster.Account()
		glog.Infof("[%v] Setting ProjectID to %v", kluster.Name, copy.Spec.Openstack.ProjectID)
	}

	var selectedRouter *openstack.Router

	if routerID := copy.Spec.Openstack.RouterID; routerID != "" {
		for _, router := range routers {
			if router.ID == routerID {
				selectedRouter = &router
				break
			}
		}
		if selectedRouter == nil {
			return fmt.Errorf("Specified router %s not found in project", routerID)
		}
	} else {
		if numRouters := len(routers); numRouters == 1 {
			selectedRouter = &routers[0]
			glog.Infof("[%v] Setting RouterID to %v", kluster.Name, selectedRouter.ID)
			copy.Spec.Openstack.RouterID = selectedRouter.ID
		} else {
			return fmt.Errorf("Found %d routers in project. Autoconfiguration not possible.", numRouters)
		}
	}

	//we have a router beyond this point
	var selectedNetwork *openstack.Network

	if networkID := copy.Spec.Openstack.NetworkID; networkID != "" {
		for _, network := range selectedRouter.Networks {
			if network.ID == networkID {
				selectedNetwork = &network
				break
			}
		}
		if selectedNetwork == nil {
			return fmt.Errorf("Selected network %s not found on router %s", networkID, selectedRouter.ID)
		}
	} else {
		if numNetworks := len(selectedRouter.Networks); numNetworks == 1 {
			selectedNetwork = &selectedRouter.Networks[0]
			copy.Spec.Openstack.NetworkID = selectedNetwork.ID
			glog.Infof("[%v] Setting NetworkID to %v", kluster.Name, selectedNetwork.ID)
		} else {
			return fmt.Errorf("Found %d networks on router %s. Auto-configuration not possible. Please choose one.", numNetworks, selectedRouter.ID)

		}
	}

	if subnetID := copy.Spec.Openstack.LBSubnetID; subnetID != "" {
		found := false
		for _, subnet := range selectedNetwork.Subnets {
			if subnet.ID == subnetID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Selected subnet %s not found in network %s", subnetID, selectedNetwork.ID)
		}
	} else {
		if numSubnets := len(selectedNetwork.Subnets); numSubnets == 1 {
			copy.Spec.Openstack.LBSubnetID = selectedNetwork.Subnets[0].ID
			glog.V(5).Infof("[%v] Setting LBSubnetID to %v", kluster.Name, copy.Spec.Openstack.LBSubnetID)
		} else {
			return fmt.Errorf("Found %d subnets for network %s. Auto-configuration not possible. Please choose one.", numSubnets, selectedNetwork.ID)
		}
	}

	if securityGroupID := copy.Spec.Openstack.SecurityGroupID; securityGroupID != "" {
		//TODO: Validate that the securitygroup id exists

	} else {
		id, err := op.Clients.Openstack.GetSecurityGroupID(kluster.Account(), "default")
		if err != nil {
			return fmt.Errorf("Failed to get id for default securitygroup in project %s: %s", err, kluster.Account())
		}
		glog.V(5).Infof("[%v] Setting SecurityGroupID to %v", kluster.Name, copy.Spec.Openstack.SecurityGroupID)
		copy.Spec.Openstack.SecurityGroupID = id
	}

	_, err = op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Update(copy)
	return err
}

//TODO: remove this after it has been deployed once everywhere, this is a poor mans migration
func (op *GroundControl) ensureSecurityGroupID(kluster *v1.Kluster) error {
	if kluster.Spec.Openstack.SecurityGroupID == "" {
		copy, err := op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Get(kluster.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		id, err := op.Clients.Openstack.GetSecurityGroupID(kluster.Account(), "default")
		if err != nil {
			return fmt.Errorf("Failed to get id for default securitygroup in project %s: %s", err, kluster.Account())
		}
		copy.Spec.Openstack.SecurityGroupID = id
		_, err = op.Clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Update(copy)
		return err
	}
	return nil
}

func (op *GroundControl) podAdd(obj interface{}) {
	pod := obj.(*api_v1.Pod)

	if klusterName, found := pod.GetLabels()["release"]; found && len(klusterName) > 0 {
		klusterKey := pod.GetNamespace() + "/" + klusterName
		glog.V(5).Infof("Pod %s added for kluster %s", pod.GetName(), klusterKey)
		op.queue.Add(klusterKey)
	}

}

func (op *GroundControl) podDelete(obj interface{}) {
	pod := obj.(*api_v1.Pod)
	if klusterName, found := pod.GetLabels()["release"]; found && len(klusterName) > 0 {
		klusterKey := pod.GetNamespace() + "/" + klusterName
		glog.V(5).Infof("Pod %s deleted for kluster %s", pod.GetName(), klusterKey)
		op.queue.Add(klusterKey)
	}
}

func (op *GroundControl) podUpdate(cur, old interface{}) {
	pod := cur.(*api_v1.Pod)
	oldPod := old.(*api_v1.Pod)
	if klusterName, found := pod.GetLabels()["release"]; found && len(klusterName) > 0 {
		if !reflect.DeepEqual(oldPod, pod) {
			klusterKey := pod.GetNamespace() + "/" + klusterName
			glog.V(5).Infof("Pod %s updated for kluster %s", pod.GetName(), klusterKey)
			op.queue.Add(klusterKey)
		}
	}
}
