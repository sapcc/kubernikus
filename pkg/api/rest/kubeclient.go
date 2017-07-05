package rest

import (
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/sapcc/kubernikus/pkg/kube"
)

var kubeconfig string

func init() {
	pflag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file with authorization information")
}

func NewKubeClient() (*kubernetes.Clientset, *rest.RESTClient, *runtime.Scheme) {
	return kube.NewClients(kube.Options{ConfigFile: kubeconfig})
}
