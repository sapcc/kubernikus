package operator

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/client"
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	"github.com/sapcc/kubernikus/pkg/openstack"
	"github.com/sapcc/kubernikus/pkg/version"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Options struct {
	kubeConfig string

	ChartDirectory string

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
	openstackClient  openstack.Client

	kubernikusInformers kubernikus_informers.SharedInformerFactory
	kubernetesInformers informers.SharedInformerFactory
}

func New(options Options) *Operator {
	kubernetesClient, err := client.NewKubernetesClient(options.kubeConfig)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes clients: %s", err)
	}

	kubernikusClient, err := client.NewKubernikusClient(options.kubeConfig)
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

	o := &Operator{
		Options:          options,
		kubernikusClient: kubernikusClient,
		kubernetesClient: kubernetesClient,
		openstackClient:  openstackClient,
	}

	o.kubernikusInformers = kubernikus_informers.NewSharedInformerFactory(o.kubernikusClient, 5*time.Minute)
	o.kubernetesInformers = informers.NewSharedInformerFactory(o.kubernetesClient, 5*time.Minute)

	o.kubernetesInformers.Core().V1().Nodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.debugAdd,
		UpdateFunc: o.debugUpdate,
		DeleteFunc: o.debugDelete,
	})

	o.kubernikusInformers.Kubernikus().V1().Klusters().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.debugAdd,
		UpdateFunc: o.debugUpdate,
		DeleteFunc: o.debugDelete,
	})

	return o
}

func (o *Operator) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	fmt.Printf("Welcome to Kubernikus %v\n", version.VERSION)

	o.kubernikusInformers.Start(stopCh)
	o.kubernetesInformers.Start(stopCh)

	o.kubernikusInformers.WaitForCacheSync(stopCh)
	o.kubernetesInformers.WaitForCacheSync(stopCh)

	glog.Info("Cache primed. Ready for Action!")
}

func (p *Operator) debugAdd(obj interface{}) {
	key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	glog.V(5).Infof("ADD %s (%s)", reflect.TypeOf(obj), key)
}

func (p *Operator) debugDelete(obj interface{}) {
	key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	glog.V(5).Infof("DELETE %s (%s)", reflect.TypeOf(obj), key)
}

func (p *Operator) debugUpdate(cur, old interface{}) {
	key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(cur)
	glog.V(5).Infof("UPDATE %s (%s)", reflect.TypeOf(cur), key)
}
