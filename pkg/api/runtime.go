package api

import (
	"github.com/go-kit/kit/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kubernikus_client_kubernetes "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers_v1 "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

type Runtime struct {
	Kubernikus           clientset.Interface
	Kubernetes           kubernetes.Interface
	Namespace            string
	Logger               log.Logger
	Images               *version.ImageRegistry
	Klusters             cache.SharedIndexInformer
	KlusterClientFactory kubernikus_client_kubernetes.SharedClientFactory
}

func (rt *Runtime) GetKluster(name string) (*v1.Kluster, error) {
	o, found, err := rt.Klusters.GetIndexer().GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.NewNotFound(v1.Resource("kluster"), name)
	}
	return o.(*v1.Kluster), nil
}

func NewRuntime(namespace string, kubernikusClient clientset.Interface, kubeClient kubernetes.Interface, logger log.Logger) *Runtime {

	informer := kubernikus_informers_v1.NewKlusterInformer(kubernikusClient, namespace, 0, cache.Indexers{})

	return &Runtime{
		Kubernetes:           kubeClient,
		Kubernikus:           kubernikusClient,
		Namespace:            namespace,
		Logger:               logger,
		Klusters:             informer,
		KlusterClientFactory: kubernikus_client_kubernetes.NewSharedClientFactory(kubeClient, informer, logger),
	}

}
