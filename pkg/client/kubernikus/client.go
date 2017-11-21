package kubernikus

import (
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
)

func NewClient(kubeconfig, context string) (clientset.Interface, error) {
	config, err := kube.NewConfig(kubeconfig, context)
	if err != nil {
		return nil, err
	}

	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
