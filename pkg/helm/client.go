package helm

import (
	"fmt"
	"runtime"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"

	"github.com/sapcc/kubernikus/pkg/helm/portforwarder"
)

func NewClient(kubeClient *kubernetes.Clientset, kubeConfig *rest.Config) (*helm.Client, error) {

	tillerHost := "tiller-deploy.kube-system"
	if _, err := rest.InClusterConfig(); err != nil {
		glog.V(2).Info("We are not running inside the cluster. Creating tunnel to tiller pod.")
		tunnel, err := portforwarder.New("kube-system", kubeClient, kubeConfig)
		if err != nil {
			return nil, err
		}
		tillerHost = fmt.Sprintf("localhost:%d", tunnel.Local)
		client := helm.NewClient(helm.Host(tillerHost))
		//Lets see how this goes: We close the tunnel as soon as the client is GC'ed.
		runtime.SetFinalizer(client, func(_ *helm.Client) {
			glog.V(2).Info("Tearing Down tunnel to tiller at %s", tillerHost)
			tunnel.Close()
		})
		return client, nil
	}
	return helm.NewClient(helm.Host(tillerHost)), nil
}
