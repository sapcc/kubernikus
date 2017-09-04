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

	kubernetes_informers "k8s.io/client-go/informers"
	kubernetes_clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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
}

type Clients struct {
	Kubernikus kubernikus_clientset.Interface
	Kubernetes kubernetes_clientset.Interface
	Openstack  openstack.Client
}

type Factories struct {
	Kubernikus kubernikus_informers.SharedInformerFactory
	Kubernetes kubernetes_informers.SharedInformerFactory
}

type KubernikusOperator struct {
	Options *KubernikusOperatorOptions
	Clients
	Factories
}

func NewKubernikusOperator(options *KubernikusOperatorOptions) *KubernikusOperator {
	var err error

	o := &KubernikusOperator{
		Options: options,
	}

	o.Clients.Kubernetes, err = kube.NewClient(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes clients: %s", err)
	}

	o.Clients.Kubernikus, err = kubernikus.NewClient(options.KubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernikus clients: %s", err)
	}

	o.Clients.Openstack, err = openstack.NewClient(
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

	o.Factories.Kubernikus = kubernikus_informers.NewSharedInformerFactory(o.Clients.Kubernikus, 5*time.Minute)
	o.Factories.Kubernetes = kubernetes_informers.NewSharedInformerFactory(o.Clients.Kubernetes, 5*time.Minute)

	o.Factories.Kubernetes.Core().V1().Nodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.debugAdd,
		UpdateFunc: o.debugUpdate,
		DeleteFunc: o.debugDelete,
	})

	o.Factories.Kubernikus.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.debugAdd,
		UpdateFunc: o.debugUpdate,
		DeleteFunc: o.debugDelete,
	})

	return o
}

func (o *KubernikusOperator) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	fmt.Printf("Welcome to Kubernikus %v\n", version.VERSION)

	o.Factories.Kubernikus.Start(stopCh)
	o.Factories.Kubernetes.Start(stopCh)

	o.Factories.Kubernikus.WaitForCacheSync(stopCh)
	o.Factories.Kubernetes.WaitForCacheSync(stopCh)

	glog.Info("Cache primed. Ready for Action!")
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
