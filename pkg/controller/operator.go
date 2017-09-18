package controller

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/meta"
	kubernetes_informers "k8s.io/client-go/informers"
	kubernetes_clientset "k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"

	helmutil "github.com/sapcc/kubernikus/pkg/client/helm"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	"github.com/sapcc/kubernikus/pkg/version"
)

type KubernikusOperatorOptions struct {
	KubeConfig string

	ChartDirectory string

	AuthURL           string
	AuthUsername      string
	AuthPassword      string
	AuthDomain        string
	AuthProject       string
	AuthProjectDomain string

	KubernikusDomain string
	Namespace        string
	Controllers      []string
}

type Clients struct {
	Kubernikus kubernikus_clientset.Interface
	Kubernetes kubernetes_clientset.Interface
	Satellites *kube.SharedClientFactory
	Openstack  openstack.Client
	Helm       *helm.Client
}

type OpenstackConfig struct {
	AuthURL           string
	AuthUsername      string
	AuthPassword      string
	AuthDomain        string
	AuthProject       string
	AuthProjectDomain string
}

type HelmConfig struct {
	ChartDirectory string
}

type KubernikusConfig struct {
	Domain      string
	Namespace   string
	Controllers map[string]Controller
}

type Config struct {
	Openstack  OpenstackConfig
	Kubernikus KubernikusConfig
	Helm       HelmConfig
}

type Factories struct {
	Kubernikus kubernikus_informers.SharedInformerFactory
	Kubernetes kubernetes_informers.SharedInformerFactory
}

type KubernikusOperator struct {
	Clients
	Config
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
		Config: Config{
			Openstack: OpenstackConfig{
				AuthURL:           options.AuthURL,
				AuthUsername:      options.AuthUsername,
				AuthPassword:      options.AuthPassword,
				AuthProject:       options.AuthProjectDomain,
				AuthProjectDomain: options.AuthProjectDomain,
			},
			Helm: HelmConfig{
				ChartDirectory: options.ChartDirectory,
			},
			Kubernikus: KubernikusConfig{
				Domain:      options.KubernikusDomain,
				Namespace:   options.Namespace,
				Controllers: make(map[string]Controller),
			},
		},
	}

	o.Clients.Kubernetes, err = kube.NewClient(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes clients: %s", err)
	}

	o.Clients.Kubernikus, err = kubernikus.NewClient(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernikus clients: %s", err)
	}

	config, err := kube.NewConfig(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes config: %s", err)
	}
	o.Clients.Helm, err = helmutil.NewClient(o.Clients.Kubernetes, config)
	if err != nil {
		glog.Fatalf("Failed to create helm client: %s", err)
	}

	o.Factories.Kubernikus = kubernikus_informers.NewSharedInformerFactory(o.Clients.Kubernikus, DEFAULT_RECONCILIATION)
	o.Factories.Kubernetes = kubernetes_informers.NewSharedInformerFactory(o.Clients.Kubernetes, DEFAULT_RECONCILIATION)

	o.Clients.Openstack = openstack.NewClient(
		o.Factories.Kubernetes,
		options.AuthURL,
		options.AuthUsername,
		options.AuthPassword,
		options.AuthDomain,
		options.AuthProject,
		options.AuthProjectDomain,
	)

	for _, k := range options.Controllers {
		switch k {
		case "groundctl":
			o.Config.Kubernikus.Controllers["groundctl"] = NewGroundController(o.Factories, o.Clients, o.Config)
		case "launchctl":
			o.Config.Kubernikus.Controllers["launchctl"] = NewLaunchController(o.Factories, o.Clients)
		case "wormholegenerator":
			o.Config.Kubernikus.Controllers["wormholegenerator"] = NewWormholeGenerator(o.Factories, o.Clients)
		}
	}

	o.Clients.Satellites = kube.NewSharedClientFactory(
		o.Clients.Kubernetes.Core().Secrets(options.Namespace),
		o.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer(),
	)

	return o
}

func (o *KubernikusOperator) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	fmt.Printf("Welcome to Kubernikus %v\n", version.VERSION)

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
		glog.Infof("Found release %v for pod %v", release, meta.GetName())
		return []string{release}, nil
	}
	glog.Infof("meta labels: %v", meta.GetLabels())
	return []string{""}, errors.New("object has no release label")

}
