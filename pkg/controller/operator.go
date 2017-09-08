package controller

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	"github.com/sapcc/kubernikus/pkg/version"

	helmutil "github.com/sapcc/kubernikus/pkg/client/helm"
	kubernetes_informers "k8s.io/client-go/informers"
	kubernetes_clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/helm/pkg/helm"
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
}

type Clients struct {
	Kubernikus kubernikus_clientset.Interface
	Kubernetes kubernetes_clientset.Interface
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
	Domain string
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
	GROUNDCTL_WORKERS       = 10
	LAUNCHCTL_WORKERS       = 1
	RECONCILIATION_DURATION = 5 * time.Minute
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
				Domain: options.KubernikusDomain,
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

	o.Factories.Kubernikus = kubernikus_informers.NewSharedInformerFactory(o.Clients.Kubernikus, RECONCILIATION_DURATION)
	o.Factories.Kubernetes = kubernetes_informers.NewSharedInformerFactory(o.Clients.Kubernetes, RECONCILIATION_DURATION)

	o.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.debugAdd,
		UpdateFunc: o.debugUpdate,
		DeleteFunc: o.debugDelete,
	})

	o.Clients.Openstack = openstack.NewClient(
		o.Factories.Kubernetes,
		options.AuthURL,
		options.AuthUsername,
		options.AuthPassword,
		options.AuthDomain,
		options.AuthProject,
		options.AuthProjectDomain,
	)

	return o
}

func (o *KubernikusOperator) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	fmt.Printf("Welcome to Kubernikus %v\n", version.VERSION)

	groundctl := NewGroundController(o.Factories, o.Clients, o.Config)
	launchctl := NewLaunchController(o.Factories, o.Clients)

	o.Factories.Kubernikus.Start(stopCh)
	o.Factories.Kubernetes.Start(stopCh)

	o.Factories.Kubernikus.WaitForCacheSync(stopCh)
	o.Factories.Kubernetes.WaitForCacheSync(stopCh)

	glog.Info("Cache primed. Ready for Action!")

	go groundctl.Run(GROUNDCTL_WORKERS, stopCh, wg)
	go launchctl.Run(LAUNCHCTL_WORKERS, stopCh, wg)
}

func (p *KubernikusOperator) debugAdd(obj interface{}) {
	key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	glog.V(5).Infof("ADD %s (%s)", reflect.TypeOf(obj), key)
}

func (p *KubernikusOperator) debugDelete(obj interface{}) {
	key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	glog.V(5).Infof("DELETE %s (%s)", reflect.TypeOf(obj), key)
}

func (p *KubernikusOperator) debugUpdate(cur, old interface{}) {
	key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(cur)
	glog.V(5).Infof("UPDATE %s (%s)", reflect.TypeOf(cur), key)
}
