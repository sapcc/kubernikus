package clusterapi 

import (
	kitlog "github.com/go-kit/kit/log"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterapi "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

func NewConfig(kubeconfig, context string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	if len(context) > 0 {
		overrides.CurrentContext = context
	}

	if len(kubeconfig) > 0 {
		rules.ExplicitPath = kubeconfig
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
}

func NewClient(kubeconfig, context string, logger kitlog.Logger) (clusterapi.Interface, error) {
	config, err := NewConfig(kubeconfig, context)
	if err != nil {
		return nil, err
	}

	clientset, err := clusterapi.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	logger.Log(
		"msg", "created new clusterapi client",
		"host", config.Host,
		"v", 3,
	)

	return clientset, nil
}
