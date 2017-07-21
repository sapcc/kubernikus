package kube

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Options struct {
	ConfigFile string
}

type ClientCache struct {
	config    *rest.Config
	clientset *kubernetes.Clientset
	tprClient *rest.RESTClient
	tprScheme *runtime.Scheme
}

func NewClientCache(options Options) (*ClientCache, error) {
	config := newClientConfig(options)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	glog.V(3).Infof("Using Kubernetes Api at %s", config.Host)
	clients := &ClientCache{
		config:    newClientConfig(options),
		clientset: clientset,
	}

	if err := clients.setupTPRClient(); err != nil {
		return nil, err
	}
	return clients, nil
}

func (c *ClientCache) Config() *rest.Config {
	return c.config
}

func (c *ClientCache) Clientset() *kubernetes.Clientset {
	return c.clientset
}

func (c *ClientCache) TPRClient() *rest.RESTClient {
	return c.tprClient
}

func (c *ClientCache) TPRScheme() *runtime.Scheme {
	return c.tprScheme
}

func (c *ClientCache) setupTPRClient() error {
	if err := ensureTPR(c.Clientset()); err != nil {
		return err
	}

	tprClient, tprScheme, err := newTPRClient(c.Config())
	if err != nil {
		return err
	}
	if err := waitForTPR(tprClient); err != nil {
		return err
	}

	c.tprClient = tprClient
	c.tprScheme = tprScheme
	return nil
}

func newClientConfig(options Options) *rest.Config {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	if options.ConfigFile != "" {
		rules.ExplicitPath = options.ConfigFile
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		glog.Fatalf("Couldn't get Kubernetes default config: %s", err)
	}

	return config
}
