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

func NewClients(options Options) (*kubernetes.Clientset, *rest.RESTClient, *runtime.Scheme) {
	config := NewClientConfig(options)
	clientset, err := NewClientset(config)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes clientset: %s", err)
	}
	if err := EnsureTPR(clientset); err != nil {
		glog.Fatalf("Failed to create TPR: %s", err)
	}

	tprClient, tprScheme, err := NewTPRClient(config)
	if err != nil {
		glog.Fatalf("Failed to create TPR client: %s", err)
	}

	glog.V(3).Info("Waiting for TPR resource to become available")
	if err := WaitForTPR(tprClient); err != nil {
		glog.Fatalf("TPR resource unavailable: %s", err)
	}
	return clientset, tprClient, tprScheme
}

func NewClientset(config *rest.Config) (*kubernetes.Clientset, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	glog.V(3).Infof("Using Kubernetes Api at %s", config.Host)
	return client, nil
}

func NewClientConfig(options Options) *rest.Config {
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
