package controller

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubernetes_informers "k8s.io/client-go/informers"
	kubernetes_clientset "k8s.io/client-go/kubernetes"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/helm/pkg/helm"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	helmutil "github.com/sapcc/kubernikus/pkg/client/helm"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	kubernikus_informers_v1 "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
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
	Domain         string
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
	//Manually create shared Kluster informer that only watches the given namespace
	o.Factories.Kubernikus.InformerFor(
		&kubernikus_v1.Kluster{},
		func(client kubernikus_clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			return kubernikus_informers_v1.NewKlusterInformer(
				client,
				options.Namespace,
				resyncPeriod,
				cache.Indexers{},
			)
		},
	)
	o.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.debugAdd,
		UpdateFunc: o.debugUpdate,
		DeleteFunc: o.debugDelete,
	})

	o.Factories.Kubernetes = kubernetes_informers.NewSharedInformerFactory(o.Clients.Kubernetes, RECONCILIATION_DURATION)
	//Manually create shared pod Informer that only watches the given namespace
	o.Factories.Kubernetes.InformerFor(&api_v1.Pod{}, func(client kubernetes_clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		return cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(o meta_v1.ListOptions) (runtime.Object, error) {
					return client.CoreV1().Pods(options.Namespace).List(o)
				},
				WatchFunc: func(o meta_v1.ListOptions) (watch.Interface, error) {
					return client.CoreV1().Pods(options.Namespace).Watch(o)
				},
			},
			&api_v1.Pod{},
			resyncPeriod,
			cache.Indexers{"kluster": MetaLabelReleaseIndexFunc},
		)
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
