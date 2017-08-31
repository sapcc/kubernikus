package client

import (
	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/generated/clientset"
)

func NewKubernikusClient(kubeconfig string) (clientset.Interface, error) {
	config, err := config(kubeconfig)
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
