package rest

import (
	"github.com/golang/glog"
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

func NewKubeClients() (kubernikus_clientset.Interface, kubernetes_clientset.Interface) {
	client, err := kubernikus.NewClient(kubeconfig, context)

	if err != nil {
		glog.Fatal("Failed to create kubernikus clients: %s", err)
	}

	kubernetesClient, err := kubernetes.NewClient(kubeconfig, context)
	if err != nil {
		glog.Fatal("Failed to create kubernetes clients: %s", err)
	}

	config, err := kubernetes.NewConfig(kubeconfig, context)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes config: %s", err)
	}
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		glog.Fatal("Failed to create apiextenstionsclient: %s", err)
	}

	if err := kubernetes.EnsureCRD(apiextensionsclientset); err != nil {
		glog.Fatalf("Couldn't create CRD: %s", err)
	}

	return client, kubernetesClient
}
