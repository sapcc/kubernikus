package kubernetes

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

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

	apiHost := k.Status.Apiserver
	var dialerFunc func(string, string) (net.Conn, error)

	// If run inside a kubernetes cluster we want to bypass the sni proxy and access the api service directly
	// if we run outside (dev) we fall back to using the fqdn that is exposed by the sni ingress controller
	// We need to provide a custom dialer to add the kluster namespace to the dns resolution because the
	// apiserver cert is missing an SAN for $kluster.$namespace
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		apiHost = fmt.Sprintf("https://%s:%d", k.Name, k.Spec.AdvertisePort)
		dialer := net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		dialerFunc = func(network, _ string) (net.Conn, error) {
			return dialer.Dial(network, fmt.Sprintf("%s.%s:%d", k.Name, k.Namespace, k.Spec.AdvertisePort))
		}
	}

	c := rest.Config{
		Host: apiHost,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: []byte(secret.ApiserverClientsClusterAdminCertificate),
			KeyData:  []byte(secret.ApiserverClientsClusterAdminPrivateKey),
			CAData:   []byte(secret.TLSCACertificate),
		},
		Dial: dialerFunc,
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
