package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	api_v1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes_informers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	helmutil "github.com/sapcc/kubernikus/pkg/client/helm"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/deorbit"
	"github.com/sapcc/kubernikus/pkg/controller/launch"
	"github.com/sapcc/kubernikus/pkg/controller/routegc"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
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
	MetricPort          int
}

type KubernikusOperator struct {
	config.Config
	config.Factories
	config.Clients
	Logger log.Logger
}

const (
	DEFAULT_RECONCILIATION = 5 * time.Minute
)

func NewKubernikusOperator(options *KubernikusOperatorOptions, logger log.Logger) (*KubernikusOperator, error) {
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
		Logger: logger,
	}

	o.Clients.Kubernetes, err = kube.NewClient(options.KubeConfig, options.Context, logger)

	if err != nil {
		return nil, fmt.Errorf("Failed to create kubernetes clients: %s", err)
	}

	o.Clients.Kubernikus, err = kubernikus.NewClient(options.KubeConfig, options.Context)
	if err != nil {
		return nil, fmt.Errorf("Failed to create kubernikus clients: %s", err)
	}

	config, err := kube.NewConfig(options.KubeConfig, options.Context)
	if err != nil {
		return nil, fmt.Errorf("Failed to create kubernetes config: %s", err)
	}
	o.Clients.Helm, err = helmutil.NewClient(o.Clients.Kubernetes, config, logger)
	if err != nil {
		return nil, fmt.Errorf("Failed to create helm client: %s", err)
	}

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create apiextenstionsclient: %s", err)
	}

	if err := kube.EnsureCRD(apiextensionsclientset, logger); err != nil {
		return nil, fmt.Errorf("Couldn't create CRD: %s", err)
	}

	o.Factories.Kubernikus = kubernikus_informers.NewFilteredSharedInformerFactory(o.Clients.Kubernikus, DEFAULT_RECONCILIATION, options.Namespace, nil)
	o.Factories.Kubernetes = kubernetes_informers.NewFilteredSharedInformerFactory(o.Clients.Kubernetes, DEFAULT_RECONCILIATION, options.Namespace, nil)

	secrets := o.Clients.Kubernetes.Core().Secrets(options.Namespace)
	klusters := o.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer()

	o.Clients.Openstack = openstack.NewClient(secrets, klusters,
		options.AuthURL,
		options.AuthUsername,
		options.AuthPassword,
		options.AuthDomain,
		options.AuthProject,
		options.AuthProjectDomain,
		logger,
	)

	o.Clients.Satellites = kube.NewSharedClientFactory(secrets, klusters, logger)

	// Add kubernikus types to the default Kubernetes Scheme so events can be
	// logged for those types.
	v1.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartEventWatcher(func(e *api_v1.Event) {
		logger.Log(
			"controller", "operator",
			"resource", "event",
			"msg", e.Message,
			"reason", e.Reason,
			"type", e.Type,
			"kind", e.InvolvedObject.Kind,
			"namespace", e.InvolvedObject.Namespace,
			"name", e.InvolvedObject.Name,
			"v", 2)
	})

	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: o.Clients.Kubernetes.CoreV1().Events(o.Config.Kubernikus.Namespace)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, api_v1.EventSource{Component: "operator"})

	for _, k := range options.Controllers {
		switch k {
		case "groundctl":
			o.Config.Kubernikus.Controllers["groundctl"] = NewGroundController(10, o.Factories, o.Clients, recorder, o.Config, logger)
		case "launchctl":
			o.Config.Kubernikus.Controllers["launchctl"] = launch.NewController(1, o.Factories, o.Clients, recorder, logger)
		case "routegc":
			o.Config.Kubernikus.Controllers["routegc"] = routegc.New(60*time.Second, o.Factories.Kubernikus.Kubernikus().V1().Klusters(), o.Clients.Openstack, logger)
		case "deorbiter":
			o.Config.Kubernikus.Controllers["deorbiter"] = deorbit.NewController(10, o.Factories, o.Clients, recorder, logger)
		}
	}

	return o, err
}

func (o *KubernikusOperator) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	o.Logger.Log(
		"msg", "starting kubernikus operator",
		"namespace", o.Config.Kubernikus.Namespace,
		"version", version.GitCommit)

	kube.WaitForServer(o.Clients.Kubernetes, stopCh, o.Logger)

	o.Factories.Kubernikus.Start(stopCh)
	o.Factories.Kubernetes.Start(stopCh)

	o.Factories.Kubernikus.WaitForCacheSync(stopCh)
	o.Factories.Kubernetes.WaitForCacheSync(stopCh)

	o.Logger.Log("msg", "Cache primed. Ready for Action!")

	for _, controller := range o.Config.Kubernikus.Controllers {
		go controller.Run(stopCh, wg)
	}
}
