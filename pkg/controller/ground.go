package controller

import (
	"fmt"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/goutils"
	"github.com/go-kit/kit/log"
	"google.golang.org/grpc"
	api_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	informers_v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/helm/pkg/helm"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	informers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util"
	helm_util "github.com/sapcc/kubernikus/pkg/util/helm"
	waitutil "github.com/sapcc/kubernikus/pkg/util/wait"
)

const (
	KLUSTER_RECHECK_INTERVAL = 5 * time.Minute

	//Reason constants for the event recorder
	ConfigurationError = "ConfigurationError"
	FailedCreate       = "FailedCreate"

	GroundctlFinalizer = "groundctl"
)

type GroundControl struct {
	config.Clients
	config.Factories
	config.Config
	Recorder record.EventRecorder

	queue           workqueue.RateLimitingInterface
	klusterInformer informers_kubernikus.KlusterInformer
	podInformer     informers_v1.PodInformer

	Logger      log.Logger
	threadiness int
}

func NewGroundController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, config config.Config, logger log.Logger) *GroundControl {
	logger = log.With(logger,
		"controller", "ground")

	operator := &GroundControl{
		Clients:         clients,
		Factories:       factories,
		Config:          config,
		Recorder:        recorder,
		queue:           workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		klusterInformer: factories.Kubernikus.Kubernikus().V1().Klusters(),
		podInformer:     factories.Kubernetes.Core().V1().Pods(),
		Logger:          logger,
		threadiness:     threadiness,
	}

	operator.klusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    operator.klusterAdd,
		UpdateFunc: operator.klusterUpdate,
		DeleteFunc: operator.klusterTerminate,
	})

	operator.podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    operator.podAdd,
		UpdateFunc: operator.podUpdate,
		DeleteFunc: operator.podDelete,
	})

	return operator
}

func (op *GroundControl) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer op.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	op.Logger.Log(
		"msg", "starting GroundControl",
		"threadiness", op.threadiness)

	for i := 0; i < op.threadiness; i++ {
		go wait.Until(op.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(KLUSTER_RECHECK_INTERVAL)
	go func() {
		for {
			select {
			case <-ticker.C:
				op.Logger.Log(
					"msg", "I now would do reconciliation if it was implemented",
					"kluster_recheck_interval", KLUSTER_RECHECK_INTERVAL,
					"v", 2)
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

	op.Logger.Log(
		"msg", "Error running handler",
		"err", err)
	op.queue.AddRateLimited(key)

	return true
}

func (op *GroundControl) updateKluster(namespace, name string, updateFunc func(kluster *v1.Kluster) error) error {
	_, err := util.UpdateKlusterWithRetries(
		op.Clients.Kubernikus.Kubernikus().Klusters(namespace),
		op.klusterInformer.Lister().Klusters(namespace),
		name,
		updateFunc)
	return err
}

func (op *GroundControl) handler(key string) error {
	obj, exists, err := op.klusterInformer.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("Failed to fetch key %s from cache: %s", key, err)
	}
	if !exists {
		op.Logger.Log(
			"msg", "kluster resource already deleted",
			"kluster", key,
			"v", 2)
	} else {
		kluster := obj.(*v1.Kluster)
		if kluster.Disabled() {
			return nil
		}
		op.Logger.Log(
			"msg", "handling kluster",
			"kluster", kluster.GetName(),
			"phase", kluster.Status.Phase,
			"project", kluster.Account(),
			"v", 5)

		metrics.SetMetricKlusterInfo(kluster.GetNamespace(), kluster.GetName(), kluster.Status.Version, kluster.Spec.Openstack.ProjectID, kluster.GetAnnotations(), kluster.GetLabels())
		metrics.SetMetricKlusterStatusPhase(kluster.GetName(), kluster.Status.Phase)

		switch phase := kluster.Status.Phase; phase {
		case models.KlusterPhasePending:
			{
				if op.requiresOpenstackInfo(kluster) {
					if err := op.updateKluster(kluster.Namespace, kluster.Name, op.discoverOpenstackInfo); err != nil {
						op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, ConfigurationError, "Discovery of openstack parameters failed: %s", err)
						return err
					}
					return nil
				}

				if op.requiresKubernikusInfo(kluster) {
					if err := op.updateKluster(kluster.Namespace, kluster.Name, op.discoverKubernikusInfo); err != nil {
						op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, ConfigurationError, "Discovery of kubernikus parameters failed: %s", err)
						return err
					}
					return nil
				}

				op.Logger.Log(
					"msg", "creating kluster",
					"kluster", kluster.GetName(),
					"project", kluster.Account(),
					"phase", kluster.Status.Phase)
				if err := op.createKluster(kluster); err != nil {
					op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, FailedCreate, "Failed to create cluster: %s", err)
					return err
				}
				if err := op.updatePhase(kluster, models.KlusterPhaseCreating, "Creating Cluster"); err != nil {
					op.Logger.Log(
						"msg", "failed to update status of kluster",
						"kluster", kluster.GetName(),
						"project", kluster.Account(),
						"err", err)
				}
				op.Logger.Log(
					"msg", "created kluster",
					"kluster", kluster.GetName(),
					"project", kluster.Account(),
					"phase", kluster.Status.Phase)
			}
		case models.KlusterPhaseCreating:
			pods, err := op.podInformer.Lister().List(labels.SelectorFromValidatedSet(map[string]string{"release": kluster.GetName()}))
			if err != nil {
				return err
			}
			podsReady := 0
			for _, pod := range pods {
				if kubernetes.IsPodReady(pod) {
					podsReady++
				}
			}
			op.Logger.Log(
				"msg", "pod readiness",
				"kluster", kluster.GetName(),
				"project", kluster.Account(),
				"expected", len(pods),
				"actual", podsReady)
			if podsReady == 4 {
				clientset, err := op.Clients.Satellites.ClientFor(kluster)
				if err != nil {
					return err
				}
				if err := ground.SeedKluster(clientset, kluster); err != nil {
					return err
				}
				if err := op.updatePhase(kluster, models.KlusterPhaseRunning, ""); err != nil {
					op.Logger.Log(
						"msg", "failed to update status of kluster",
						"kluster", kluster.GetName(),
						"project", kluster.Account(),
						"err", err)
				}
				metrics.SetMetricBootDurationSummary(kluster.GetCreationTimestamp().Time, time.Now())
				op.Logger.Log(
					"msg", "kluster is ready",
					"kluster", kluster.GetName(),
					"project", kluster.Account())
			}
		case models.KlusterPhaseTerminating:
			{
				// Wait until all other finalizers are done.
				//
				// Groundctl needs to be last because it deletes the API machinery, which is
				// needed for cleanup of Openstack resources, like Volumes, LBs, Routes.
				// Additionally, this also removes the Secret and ServiceUsers. Without them
				// clean-up is impossiple.
				//
				// There's a "soft" agreement that Finalizers are executed in order from
				// first to last. Here we check that Groundctl is the last remaining one and
				// spare us the trouble to maintain a ordered list.
				if !(len(kluster.Finalizers) == 1 && kluster.Finalizers[0] == GroundctlFinalizer) {
					return nil
				}

				op.Logger.Log(
					"msg", "terminating kluster",
					"kluster", kluster.GetName(),
					"project", kluster.Account())
				if err := op.terminateKluster(kluster); err != nil {
					op.Recorder.Eventf(kluster, api_v1.EventTypeWarning, "", "Failed to terminate cluster: %s", err)
					op.Logger.Log(
						"msg", "Failed to terminate kluster",
						"kluster", kluster.GetName(),
						"project", kluster.Account(),
						"err", err)
					return err
				}
				metrics.SetMetricKlusterTerminated(kluster.GetName())
				op.Logger.Log(
					"msg", "terminated kluster",
					"kluster", kluster.GetName(),
					"project", kluster.Account())
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
	op.Logger.Log(
		"msg", "Added kluster resource",
		"key", key)
	op.queue.Add(key)
}

func (op *GroundControl) klusterTerminate(obj interface{}) {
	c := obj.(*v1.Kluster)
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(c)
	if err != nil {
		return
	}
	op.Logger.Log(
		"msg", "Deleted kluster resource",
		"key", key)
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
		op.Logger.Log(
			"msg", "Updated kluster resource",
			"key", key)
		op.queue.Add(key)
	}
}

func (op *GroundControl) updatePhase(kluster *v1.Kluster, phase models.KlusterPhase, message string) error {

	//Do nothing is the phase is not changing
	if kluster.Status.Phase == phase {
		return nil
	}
	err := util.UpdateKlusterPhase(op.Clients.Kubernikus.Kubernikus(), kluster, phase)

	if err == nil {
		op.Recorder.Eventf(kluster, api_v1.EventTypeNormal, string(phase), "%s kluster", phase)
		kluster.Status.Message = message
		kluster.Status.Phase = phase
		//Wait for up to 5 seconds for the local cache to reflect the phase change
		waitutil.WaitForKluster(kluster, op.klusterInformer.Informer().GetIndexer(), func(k *v1.Kluster) (bool, error) {
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

	apiURL, err := op.Clients.OpenstackAdmin.GetKubernikusCatalogEntry()
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
	region, err := op.Clients.OpenstackAdmin.GetRegion()
	if err != nil {
		return err
	}

	if err := util.EnsureFinalizerCreated(op.Clients.Kubernikus.KubernikusV1(), op.klusterInformer.Lister(), kluster, GroundctlFinalizer); err != nil {
		return err
	}

	op.Logger.Log(
		"msg", "creating service user",
		"username", username,
		"kluster", kluster.GetName(),
		"project", kluster.Account())

	if err := op.Clients.OpenstackAdmin.CreateKlusterServiceUser(
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

	op.Logger.Log(
		"msg", "Installing helm release",
		"kluster", kluster.GetName(),
		"project", kluster.Account())

	op.Logger.Log(
		"msg", "Debug Chart Values",
		"values", string(rawValues),
		"v", 6)

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

		op.Logger.Log(
			"msg", "Deleting openstack user",
			"kluster", kluster.GetName(),
			"project", kluster.Account(),
			"username", username,
			"domain", domain)

		if err := op.Clients.OpenstackAdmin.DeleteUser(username, domain); err != nil {
			return err
		}
	}

	op.Logger.Log(
		"msg", "Deleting helm release",
		"kluster", kluster.GetName(),
		"project", kluster.Account())

	_, err := op.Clients.Helm.DeleteRelease(kluster.GetName(), helm.DeletePurge(true))
	if err != nil && !strings.Contains(grpc.ErrorDesc(err), fmt.Sprintf(`release: "%s" not found`, kluster.GetName())) {
		return err
	}

	if err := util.EnsureFinalizerRemoved(op.Clients.Kubernikus.KubernikusV1(), op.klusterInformer.Lister(), kluster, GroundctlFinalizer); err != nil {
		return err
	}

	// TODO: remove if all control-planes are running k8s 1.8+
	// There's a bug in the garbage-collector regarding CRDs in 1.7. It will not delete
	// the CRD even though all Finalizers are gone. As a workaround, here we try to just
	// delte the kluster again.
	//
	// This can be removed once the control-planes include garbage collector fixes
	// for CDRs (1.8+)
	//
	// See: https://github.com/kubernetes/kubernetes/issues/50528
	err = op.Clients.Kubernikus.Discovery().RESTClient().Delete().AbsPath("apis/kubernikus.sap.cc/v1").
		Namespace(kluster.Namespace).
		Resource("klusters").
		Name(kluster.Name).
		Do().
		Error()

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	waitutil.WaitForKlusterDeletion(kluster, op.klusterInformer.Informer().GetIndexer())
	return nil
}

func (op *GroundControl) requiresOpenstackInfo(kluster *v1.Kluster) bool {
	return kluster.Spec.Openstack.ProjectID == "" ||
		kluster.Spec.Openstack.RouterID == "" ||
		kluster.Spec.Openstack.NetworkID == "" ||
		kluster.Spec.Openstack.LBSubnetID == "" ||
		kluster.Spec.Openstack.LBFloatingNetworkID == ""
}

func (op *GroundControl) requiresKubernikusInfo(kluster *v1.Kluster) bool {
	return kluster.Status.Apiserver == "" || kluster.Status.Wormhole == ""
}

func (op *GroundControl) discoverKubernikusInfo(kluster *v1.Kluster) error {
	op.Logger.Log(
		"msg", "discovering KubernikusInfo",
		"kluster", kluster.GetName(),
		"project", kluster.Account(),
		"v", 5)

	if kluster.Status.Apiserver == "" {
		kluster.Status.Apiserver = fmt.Sprintf("https://%s.%s", kluster.GetName(), op.Config.Kubernikus.Domain)
		op.Logger.Log(
			"msg", "discovered ServerURL",
			"url", kluster.Status.Apiserver,
			"kluster", kluster.GetName(),
			"project", kluster.Account())
	}

	if kluster.Status.Wormhole == "" {
		kluster.Status.Wormhole = fmt.Sprintf("https://%s-wormhole.%s", kluster.GetName(), op.Config.Kubernikus.Domain)
		op.Logger.Log(
			"msg", "discovered WormholeURL",
			"url", kluster.Status.Wormhole,
			"kluster", kluster.GetName(),
			"project", kluster.Account())
	}

	return nil
}

func (op *GroundControl) discoverOpenstackInfo(kluster *v1.Kluster) error {
	op.Logger.Log(
		"msg", "discovering OpenstackInfo",
		"kluster", kluster.GetName(),
		"project", kluster.Account(),
		"v", 5)

	if kluster.Spec.Openstack.ProjectID == "" {
		kluster.Spec.Openstack.ProjectID = kluster.Account()
		op.Logger.Log(
			"msg", "discovered ProjectID",
			"id", kluster.Spec.Openstack.ProjectID,
			"kluster", kluster.GetName(),
			"project", kluster.Account())
	}

	client, err := op.Factories.Openstack.ProjectAdminClientFor(kluster.Spec.Openstack.ProjectID)
	if err != nil {
		return err
	}

	metadata, err := client.GetMetadata()
	if err != nil {
		return err
	}

	var selectedRouter *models.Router
	if routerID := kluster.Spec.Openstack.RouterID; routerID != "" {
		for _, router := range metadata.Routers {
			if router.ID == routerID {
				selectedRouter = router
				break
			}
		}
		if selectedRouter == nil {
			return fmt.Errorf("Specified router %s not found in project", routerID)
		}
	} else {
		if numRouters := len(metadata.Routers); numRouters == 1 {
			selectedRouter = metadata.Routers[0]
			op.Logger.Log(
				"msg", "discovered RouterID",
				"id", selectedRouter.ID,
				"kluster", kluster.GetName(),
				"project", kluster.Account())
			kluster.Spec.Openstack.RouterID = selectedRouter.ID
		} else {
			return fmt.Errorf("Found %d routers in project. Auto-configuration not possible.", numRouters)
		}
	}

	//we have a router beyond this point
	var selectedNetwork *models.Network
	if networkID := kluster.Spec.Openstack.NetworkID; networkID != "" {
		for _, network := range selectedRouter.Networks {
			if network.ID == networkID {
				selectedNetwork = network
				break
			}
		}
		if selectedNetwork == nil {
			return fmt.Errorf("Selected network %s not found on router %s", networkID, selectedRouter.ID)
		}
	} else {
		if numNetworks := len(selectedRouter.Networks); numNetworks == 1 {
			selectedNetwork = selectedRouter.Networks[0]
			kluster.Spec.Openstack.NetworkID = selectedNetwork.ID
			op.Logger.Log(
				"msg", "discovered NetworkID",
				"id", selectedNetwork.ID,
				"kluster", kluster.GetName(),
				"project", kluster.Account())
		} else {
			return fmt.Errorf("Found %d networks on router %s. Auto-configuration not possible. Please choose one.", numNetworks, selectedRouter.ID)

		}
	}

	if subnetID := kluster.Spec.Openstack.LBSubnetID; subnetID != "" {
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
			kluster.Spec.Openstack.LBSubnetID = selectedNetwork.Subnets[0].ID
			op.Logger.Log(
				"msg", "discovered LBSubnetID",
				"id", kluster.Spec.Openstack.LBSubnetID,
				"kluster", kluster.GetName(),
				"project", kluster.Account())
		} else {
			return fmt.Errorf("Found %d subnets for network %s. Auto-configuration not possible. Please choose one.", numSubnets, selectedNetwork.ID)
		}
	}

	if floatingNetworkID := kluster.Spec.Openstack.LBFloatingNetworkID; floatingNetworkID != "" {
		if selectedRouter.ExternalNetworkID != "" && floatingNetworkID != selectedRouter.ExternalNetworkID {
			return fmt.Errorf("External network missmatch. Router is configured with %s but config specifies %s", selectedRouter.ExternalNetworkID, floatingNetworkID)
		}
	} else {
		if selectedRouter.ExternalNetworkID == "" {
			return fmt.Errorf("Selected router %s doesn't have an external network ID set", selectedRouter.ID)
		} else {
			kluster.Spec.Openstack.LBFloatingNetworkID = selectedRouter.ExternalNetworkID
			op.Logger.Log(
				"msg", "discovered LBFloatingNetworkID",
				"id", kluster.Spec.Openstack.LBFloatingNetworkID,
				"kluster", kluster.GetName(),
				"project", kluster.Account())
		}
	}

	if secGroupName := kluster.Spec.Openstack.SecurityGroupName; secGroupName != "" {
		found := false
		for _, sg := range metadata.SecurityGroups {
			if sg.Name == secGroupName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Selected security group %s not found in project", secGroupName)
		}
	} else {
		kluster.Spec.Openstack.SecurityGroupName = "default"
		op.Logger.Log(
			"msg", "discovered SecurityGroup",
			"name", "default",
			"kluster", kluster.GetName(),
			"project", kluster.Account())
	}

	return nil
}

func (op *GroundControl) podAdd(obj interface{}) {
	pod := obj.(*api_v1.Pod)

	if klusterName, found := pod.GetLabels()["release"]; found && len(klusterName) > 0 {
		klusterKey := pod.GetNamespace() + "/" + klusterName
		op.Logger.Log(
			"msg", "pod added",
			"name", pod.GetName(),
			"kluster", klusterKey,
			"v", 5)
		op.queue.Add(klusterKey)
	}

}

func (op *GroundControl) podDelete(obj interface{}) {
	pod := obj.(*api_v1.Pod)
	if klusterName, found := pod.GetLabels()["release"]; found && len(klusterName) > 0 {
		klusterKey := pod.GetNamespace() + "/" + klusterName
		op.Logger.Log(
			"msg", "pod deleted",
			"name", pod.GetName(),
			"kluster", klusterKey,
			"v", 5)
		op.queue.Add(klusterKey)
	}
}

func (op *GroundControl) podUpdate(cur, old interface{}) {
	pod := cur.(*api_v1.Pod)
	oldPod := old.(*api_v1.Pod)
	if klusterName, found := pod.GetLabels()["release"]; found && len(klusterName) > 0 {
		if !reflect.DeepEqual(oldPod, pod) {
			klusterKey := pod.GetNamespace() + "/" + klusterName
			op.Logger.Log(
				"msg", "pod updated",
				"name", pod.GetName(),
				"kluster", klusterKey,
				"v", 5)
			op.queue.Add(klusterKey)
		}
	}
}
