package rest

import (
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	kubernetes_clientset "k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
)

var kubeconfig string
var namespace string

func init() {
	pflag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file with authorization information")
	pflag.StringVar(&namespace, "namespace", "kubernikus", "Namespace the apiserver should work in")
}

func NewKubeClients() (kubernikus_clientset.Interface, kubernetes_clientset.Interface) {
	client, err := kubernikus.NewClient(kubeconfig)

	if err != nil {
		glog.Fatal("Failed to create kubernikus clients: %s", err)
	}

	kubernetesClient, err := kubernetes.NewClient(kubeconfig)
	if err != nil {
		glog.Fatal("Failed to create kubernetes clients: %s", err)
	}

	return client, kubernetesClient
}
