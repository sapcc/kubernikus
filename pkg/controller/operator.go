package controller

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubernetes_informers "k8s.io/client-go/informers"
	kubernetes_clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/helm/pkg/helm"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	helmutil "github.com/sapcc/kubernikus/pkg/client/helm"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	kubernikus_informers_v1 "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

type KubernikusOperatorOptions struct {
	KubeConfig string
	Context    string

	ChartDirectory string

	AuthURL           string
	AuthUsername      string
	AuthPassword      string
	AuthDomain        string
	AuthProject       string
	AuthProjectDomain string

	KubernikusDomain    string
	KubernikusProjectID string
	KubernikusNetworkID string
	Namespace           string
	Controllers         []string
}

type Clients struct {
	Kubernikus kubernikus_clientset.Interface
	Kubernetes kubernetes_clientset.Interface
	Satellites *kube.SharedClientFactory
	Openstack  openstack.Client
	Helm       *helm.Client
}

type Factories struct {
	Kubernikus kubernikus_informers.SharedInformerFactory
	Kubernetes kubernetes_informers.SharedInformerFactory
}

type KubernikusOperator struct {
	Clients
	config.Config
	Factories
}

const (
	DEFAULT_WORKERS        = 1
	DEFAULT_RECONCILIATION = 5 * time.Minute
)

var (
	CONTROLLER_OPTIONS = map[string]int{
		"groundctl":         10,
		"launchctl":         DEFAULT_WORKERS,
		"wormholegenerator": DEFAULT_WORKERS,
	}
)

func NewKubernikusOperator(options *KubernikusOperatorOptions) *KubernikusOperator {
	var err error

	o := &KubernikusOperator{
		Config: config.Config{
			Openstack: config.OpenstackConfig{
				AuthURL:           options.AuthURL,
				AuthUsername:      options.AuthUsername,
				AuthPassword:      options.AuthPassword,
				AuthProject:       options.AuthProjectDomain,
				AuthProjectDomain: options.AuthProjectDomain,
			},
			Helm: config.HelmConfig{
				ChartDirectory: options.ChartDirectory,
			},
			Kubernikus: config.KubernikusConfig{
				Domain:      options.KubernikusDomain,
				Namespace:   options.Namespace,
				ProjectID:   options.KubernikusProjectID,
				NetworkID:   options.KubernikusNetworkID,
				Controllers: make(map[string]config.Controller),
			},
		},
	}

	o.Clients.Kubernetes, err = kube.NewClient(options.KubeConfig, options.Context)

	if err != nil {
		glog.Fatalf("Failed to create kubernetes clients: %s", err)
	}

	o.Clients.Kubernikus, err = kubernikus.NewClient(options.KubeConfig, options.Context)
	if err != nil {
		glog.Fatalf("Failed to create kubernikus clients: %s", err)
	}

	config, err := kube.NewConfig(options.KubeConfig, options.Context)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes config: %s", err)
	}
	o.Clients.Helm, err = helmutil.NewClient(o.Clients.Kubernetes, config)
	if err != nil {
		glog.Fatalf("Failed to create helm client: %s", err)
	}

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		glog.Fatal("Failed to create apiextenstionsclient: %s", err)
	}

	if err := kube.EnsureCRD(apiextensionsclientset); err != nil {
		glog.Fatalf("Couldn't create CRD: %s", err)
	}

	o.Factories.Kubernikus = kubernikus_informers.NewSharedInformerFactory(o.Clients.Kubernikus, DEFAULT_RECONCILIATION)
	o.Factories.Kubernetes = kubernetes_informers.NewSharedInformerFactory(o.Clients.Kubernetes, DEFAULT_RECONCILIATION)
	o.initializeCustomInformers()

	secrets := o.Clients.Kubernetes.Core().Secrets(options.Namespace)
	klusters := o.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer()

	o.Clients.Openstack = openstack.NewClient(secrets, klusters,
		options.AuthURL,
		options.AuthUsername,
		options.AuthPassword,
		options.AuthDomain,
		options.AuthProject,
		options.AuthProjectDomain,
	)

	o.Clients.Satellites = kube.NewSharedClientFactory(secrets, klusters)

	// Add kubernikus types to the default Kubernetes Scheme so events can be
	// logged for those types.
	v1.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: o.Clients.Kubernetes.CoreV1().Events(o.Config.Kubernikus.Namespace)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, api_v1.EventSource{Component: "operator"})

	for _, k := range options.Controllers {
		switch k {
		case "groundctl":
			o.Config.Kubernikus.Controllers["groundctl"] = NewGroundController(o.Factories, o.Clients, recorder, o.Config)
		case "launchctl":
			o.Config.Kubernikus.Controllers["launchctl"] = NewLaunchController(o.Factories, o.Clients)
		}
	}

	return o
}

func (o *KubernikusOperator) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	fmt.Printf("Welcome to Kubernikus %v\n", version.GitCommit)

	kube.WaitForServer(o.Clients.Kubernetes, stopCh)

	o.Factories.Kubernikus.Start(stopCh)
	o.Factories.Kubernetes.Start(stopCh)

	o.Factories.Kubernikus.WaitForCacheSync(stopCh)
	o.Factories.Kubernetes.WaitForCacheSync(stopCh)

	glog.Info("Cache primed. Ready for Action!")

	for name, controller := range o.Config.Kubernikus.Controllers {
		go controller.Run(CONTROLLER_OPTIONS[name], stopCh, wg)
	}
}

// MetaLabelReleaseIndexFunc is a default index function that indexes based on an object's release label
func MetaLabelReleaseIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	if release, found := meta.GetLabels()["release"]; found {
		glog.V(6).Infof("Found release %v for pod %v", release, meta.GetName())
		return []string{release}, nil
	}
	glog.V(6).Infof("meta labels: %v", meta.GetLabels())
	return []string{""}, errors.New("object has no release label")
}

func (o *KubernikusOperator) initializeCustomInformers() {
	//Manually create shared Kluster informer that only watches the given namespace
	o.Factories.Kubernikus.InformerFor(
		&v1.Kluster{},
		func(client kubernikus_clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			return kubernikus_informers_v1.NewKlusterInformer(
				client,
				o.Config.Kubernikus.Namespace,
				resyncPeriod,
				cache.Indexers{},
			)
		},
	)

	//Manually create shared pod Informer that only watches the given namespace
	o.Factories.Kubernetes.InformerFor(&api_v1.Pod{}, func(client kubernetes_clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		return cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(opt metav1.ListOptions) (runtime.Object, error) {
					return client.CoreV1().Pods(o.Config.Kubernikus.Namespace).List(opt)
				},
				WatchFunc: func(opt metav1.ListOptions) (watch.Interface, error) {
					return client.CoreV1().Pods(o.Config.Kubernikus.Namespace).Watch(opt)
				},
			},
			&api_v1.Pod{},
			resyncPeriod,
			cache.Indexers{"kluster": MetaLabelReleaseIndexFunc},
		)
	})
}
