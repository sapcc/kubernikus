package v1

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/generated/clientset/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type KubernikusV1Interface interface {
	RESTClient() rest.Interface
	KlustersGetter
	SAPCCloudProviderConfigsGetter
}

// KubernikusV1Client is used to interact with features provided by the kubernikus.sap.cc group.
type KubernikusV1Client struct {
	restClient rest.Interface
}

func (c *KubernikusV1Client) Klusters(namespace string) KlusterInterface {
	return newKlusters(c, namespace)
}

func (c *KubernikusV1Client) SAPCCloudProviderConfigs(namespace string) SAPCCloudProviderConfigInterface {
	return newSAPCCloudProviderConfigs(c, namespace)
}

// NewForConfig creates a new KubernikusV1Client for the given config.
func NewForConfig(c *rest.Config) (*KubernikusV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &KubernikusV1Client{client}, nil
}

// NewForConfigOrDie creates a new KubernikusV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *KubernikusV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new KubernikusV1Client for the given RESTClient.
func New(c rest.Interface) *KubernikusV1Client {
	return &KubernikusV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *KubernikusV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
