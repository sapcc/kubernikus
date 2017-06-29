package rest

import (
	"github.com/golang/glog"
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
	config := kube.NewClientConfig(kube.Options{ConfigFile: kubeconfig})
	clientset, err := kube.NewClient(config)
	if err != nil {
		glog.Fatalf("Failed to create kubernetes clientset: %s", err)
	}
	if err := kube.EnsureTPR(clientset); err != nil {
		glog.Fatalf("Failed to create TPR: %s", err)
	}

	tprClient, tprScheme, err := kube.NewTPRClient(config)
	if err != nil {
		glog.Fatalf("Failed to create TPR client: %s", err)
	}

	glog.V(3).Info("Waiting for TPR resource to become available")
	if err := kube.WaitForTPR(tprClient); err != nil {
		glog.Fatalf("TPR resource unavailable: %s", err)
	}
	return clientset, tprClient, tprScheme
}
