package api

import (
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	kubernikus_client_kubernetes "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers_v1 "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
	kubernikus_listers_v1 "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

type Runtime struct {
	Kubernikus           clientset.Interface
	Kubernetes           kubernetes.Interface
	Namespace            string
	Logger               log.Logger
	Images               *version.ImageRegistry
	KlusterClientFactory kubernikus_client_kubernetes.SharedClientFactory
	Informer             cache.SharedIndexInformer
	Klusters             kubernikus_listers_v1.KlusterLister
}

func NewRuntime(namespace string, kubernikusClient clientset.Interface, kubeClient kubernetes.Interface, logger log.Logger) *Runtime {

	informer := kubernikus_informers_v1.NewKlusterInformer(kubernikusClient, namespace, 0, cache.Indexers{})

	return &Runtime{
		Kubernetes:           kubeClient,
		Kubernikus:           kubernikusClient,
		Namespace:            namespace,
		Logger:               logger,
		KlusterClientFactory: kubernikus_client_kubernetes.NewSharedClientFactory(kubeClient, informer, logger),
		Informer:             informer,
		Klusters:             kubernikus_listers_v1.NewKlusterLister(informer.GetIndexer()),
	}

}
