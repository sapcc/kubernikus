package kubernikus

import (
	"github.com/golang/glog"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
)

func NewClient(kubeconfig string) (clientset.Interface, error) {
	config, err := kube.NewConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	glog.V(3).Infof("Using Kubernetes Api at %s", config.Host)

	return clientset, nil
}
