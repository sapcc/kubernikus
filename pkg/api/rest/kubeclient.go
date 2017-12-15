package rest

import (
	"fmt"

	kitlog "github.com/go-kit/kit/log"
	"github.com/spf13/pflag"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubernetes_clientset "k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
)

var kubeconfig string
var context string

func init() {
	pflag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file with authorization information")
	pflag.StringVar(&context, "context", "", "Override context")
}

func NewKubeClients(logger kitlog.Logger) (kubernikus_clientset.Interface, kubernetes_clientset.Interface, error) {
	client, err := kubernikus.NewClient(kubeconfig, context)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create kubernikus clients: %s", err)
	}

	kubernetesClient, err := kubernetes.NewClient(kubeconfig, context, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create kubernetes clients: %s", err)
	}

	config, err := kubernetes.NewConfig(kubeconfig, context)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create kubernetes config: %s", err)
	}
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create apiextenstionsclient: %s", err)
	}

	if err := kubernetes.EnsureCRD(apiextensionsclientset, logger); err != nil {
		return nil, nil, fmt.Errorf("Couldn't create CRD: %s", err)
	}

	return client, kubernetesClient, nil
}
