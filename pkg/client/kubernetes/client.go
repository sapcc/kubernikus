package kubernetes

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type SharedClientFactory struct {
	clients          *sync.Map
	secretsInterface typedv1.SecretInterface
}

func NewSharedClientFactory(secrets typedv1.SecretInterface, klusterEvents cache.SharedIndexInformer) *SharedClientFactory {
	factory := &SharedClientFactory{
		clients:          new(sync.Map),
		secretsInterface: secrets,
	}

	if klusterEvents != nil {
		klusterEvents.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				if kluster, ok := obj.(*kubernikus_v1.Kluster); ok {
					glog.V(5).Info("Deleting shared kubernetes client for kluster %s", kluster.GetName())
					factory.clients.Delete(kluster.GetUID())
				}
			},
		})
	}

	return factory
}

func (f *SharedClientFactory) ClientFor(k *kubernikus_v1.Kluster) (kubernetes.Interface, error) {
	if client, found := f.clients.Load(k.GetUID()); found {
		return client.(kubernetes.Interface), nil
	}
	glog.V(5).Info("Creating new shared kubernetes client for kluster %s", k.GetName())
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
		Host: k.Spec.KubernikusInfo.ServerURL,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: clientCert,
			KeyData:  clientKey,
			CAData:   apiserverCACert,
		},
	}

	clientset, err := kubernetes.NewForConfig(&c)
	if err != nil {
		return nil, err
	}

	f.clients.Store(k.GetUID(), clientset)
	return clientset, nil

}

func NewConfig(kubeconfig, context string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context}

	if len(kubeconfig) > 0 {
		rules.ExplicitPath = kubeconfig
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		glog.Fatalf("Couldn't get Kubernetes default config: %s", err)
	}

	return config, nil
}

func NewClient(kubeconfig, context string) (kubernetes.Interface, error) {
	config, err := NewConfig(kubeconfig, context)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	glog.V(3).Infof("Using Kubernetes Api at %s", config.Host)

	return clientset, nil
}

func NewClientConfigV1(name, user, url string, key, cert, ca []byte) clientcmdapiv1.Config {
	return clientcmdapiv1.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: name,
		Clusters: []clientcmdapiv1.NamedCluster{
			clientcmdapiv1.NamedCluster{
				Name: name,
				Cluster: clientcmdapiv1.Cluster{
					Server: url,
					CertificateAuthorityData: ca,
				},
			},
		},
		Contexts: []clientcmdapiv1.NamedContext{
			clientcmdapiv1.NamedContext{
				Name: name,
				Context: clientcmdapiv1.Context{
					Cluster:  name,
					AuthInfo: user,
				},
			},
		},
		AuthInfos: []clientcmdapiv1.NamedAuthInfo{
			clientcmdapiv1.NamedAuthInfo{
				Name: user,
				AuthInfo: clientcmdapiv1.AuthInfo{
					ClientCertificateData: cert,
					ClientKeyData:         key,
				},
			},
		},
	}
}

func EnsureTPR(clientset kubernetes.Interface) error {
	tpr := &v1beta1.ThirdPartyResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kluster." + kubernikus_v1.GroupName,
		},
		Versions: []v1beta1.APIVersion{
			{Name: v1.SchemeGroupVersion.Version},
		},
		Description: "Managed kubernetes cluster",
	}

	_, err := clientset.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func WaitForTPR(clientset kubernetes.Interface) error {
	return wait.Poll(100*time.Millisecond, 30*time.Second, func() (bool, error) {
		_, err := clientset.ExtensionsV1beta1().ThirdPartyResources().Get("kluster."+kubernikus_v1.GroupName, metav1.GetOptions{})
		if err == nil {
			return true, nil
		}
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	})
}

func WaitForServer(client kubernetes.Interface, stopCh <-chan struct{}) error {
	var healthzContent string

	err := wait.PollUntil(time.Second, func() (bool, error) {
		healthStatus := 0
		resp := client.Discovery().RESTClient().Get().AbsPath("/healthz").Do().StatusCode(&healthStatus)
		if healthStatus != http.StatusOK {
			glog.Errorf("Server isn't healthy yet.  Waiting a little while.")
			return false, nil
		}
		content, _ := resp.Raw()
		healthzContent = string(content)

		return true, nil
	}, stopCh)
	if err != nil {
		return fmt.Errorf("Failed to contact apiserver. Last health: %v  Error: %v", healthzContent, err)
	}

	return nil
}
