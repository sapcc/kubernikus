package kubernetes

import (
	"errors"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type SharedClientFactory interface {
	ClientFor(k *kubernikus_v1.Kluster) (clientset kubernetes.Interface, err error)
}

type sharedClientFactory struct {
	clients          *sync.Map
	secretsInterface typedv1.SecretInterface
	Logger           kitlog.Logger
}

func NewSharedClientFactory(secrets typedv1.SecretInterface, klusterEvents cache.SharedIndexInformer, logger kitlog.Logger) SharedClientFactory {
	factory := &sharedClientFactory{
		clients:          new(sync.Map),
		secretsInterface: secrets,
		Logger:           kitlog.With(logger, "client", "kubernetes"),
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
	secret, err := f.secretsInterface.Get(k.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	apiserverCACert, ok := secret.Data["tls-ca.pem"]
	if !ok {
		return nil, errors.New("tls-ca.pem not found in secret")
	}
	clientCert, ok := secret.Data["apiserver-clients-cluster-admin.pem"]
	if !ok {
		return nil, errors.New("tls-ca.pem not found in secret")
	}
	clientKey, ok := secret.Data["apiserver-clients-cluster-admin-key.pem"]
	if !ok {
		return nil, errors.New("tls-ca.pem not found in secret")
	}

	c := rest.Config{
		Host: k.Status.Apiserver,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: clientCert,
			KeyData:  clientKey,
			CAData:   apiserverCACert,
		},
	}

	clientset, err = kubernetes.NewForConfig(&c)
	if err != nil {
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
