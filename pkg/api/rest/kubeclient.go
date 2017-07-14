package rest

import (
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/sapcc/kubernikus/pkg/kube"
)

var kubeconfig string

func init() {
	pflag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file with authorization information")
}

func NewKubeClients() *kube.ClientCache {
	clients, err := kube.NewClientCache(kube.Options{ConfigFile: kubeconfig})
	if err != nil {
		glog.Fatal("Failed to create kubernetes clients: %s", err)
	}
	return clients
}
