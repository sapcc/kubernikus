package rest

import (
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus"
)

var kubeconfig string

func init() {
	pflag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file with authorization information")
}

func NewKubeClients() *api.Clients {
	client, err := kubernikus.NewClient(kubeconfig)

	if err != nil {
		glog.Fatal("Failed to create kubernikus clients: %s", err)
	}

	kubernetesClient, err := kubernetes.NewClient(kubeconfig)
	if err != nil {
		glog.Fatal("Failed to create kubernetes clients: %s", err)
	}

	return &api.Clients{
		Kubernikus: client,
		Kubernetes: kubernetesClient,
	}
}
