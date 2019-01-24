package kubernetes

import (
	"sync"

	kitlog "github.com/go-kit/kit/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util"
)

type SharedClientFactory interface {
	ClientFor(k *kubernikus_v1.Kluster) (clientset kubernetes.Interface, err error)
}

type sharedClientFactory struct {
	clients         *sync.Map
	clientInterface kubernetes.Interface
	Logger          kitlog.Logger
}

func NewSharedClientFactory(client kubernetes.Interface, klusterEvents cache.SharedIndexInformer, logger kitlog.Logger) SharedClientFactory {
	factory := &sharedClientFactory{
		clients:         new(sync.Map),
		clientInterface: client,
		Logger:          kitlog.With(logger, "client", "kubernetes"),
	}

	if klusterEvents != nil {
		klusterEvents.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if kluster, ok := obj.(*kubernikus_v1.Kluster); ok {
					factory.clients.Delete(kluster.GetUID())
					factory.Logger.Log(
						"msg", "deleted shared kubernetes client",
						"kluster", kluster.GetName(),
						"project", kluster.Account(),
						"v", 2,
					)
				}
			},
		})
	}

	return factory
}

func (f *sharedClientFactory) ClientFor(k *kubernikus_v1.Kluster) (clientset kubernetes.Interface, err error) {
	defer func() {
		f.Logger.Log(
			"msg", "created shared kubernetes client",
			"kluster", k.GetName(),
			"project", k.Account(),
			"v", 2,
			"err", err,
		)
	}()

	if client, found := f.clients.Load(k.GetUID()); found {
		return client.(kubernetes.Interface), nil
	}

	secret, err := util.KlusterSecret(f.clientInterface, k)
	if err != nil {
		return nil, err
	}

	c := rest.Config{
		Host: k.Status.Apiserver,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: []byte(secret.ApiserverClientsClusterAdminCertificate),
			KeyData:  []byte(secret.ApiserverClientsClusterAdminPrivateKey),
			CAData:   []byte(secret.TLSCACertificate),
		},
	}

	clientset, err = kubernetes.NewForConfig(&c)
	if err != nil {
		return nil, err
	}
	//Ensure the client can actually talk to before saving it to the cache
	if _, err := clientset.Discovery().ServerVersion(); err != nil {
		return nil, err
	}

	f.clients.Store(k.GetUID(), clientset)
	return clientset, nil

}

type MockSharedClientFactory struct {
	Clientset kubernetes.Interface
}

func (m *MockSharedClientFactory) ClientFor(k *kubernikus_v1.Kluster) (kubernetes.Interface, error) {
	return m.Clientset, nil

}
