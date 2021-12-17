package config

import (
	"fmt"
	"sync"

	"github.com/go-kit/kit/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	kubernetes_informers "k8s.io/client-go/informers"
	kubernetes_clientset "k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
	kubernikus_clientset "github.com/sapcc/kubernikus/pkg/generated/clientset"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	"github.com/sapcc/kubernikus/pkg/version"
)

type Controller interface {
	Run(stopCh <-chan struct{}, wg *sync.WaitGroup)
}

type OpenstackConfig struct {
	AuthURL           string
	AuthUsername      string
	AuthPassword      string
	AuthDomain        string
	AuthProject       string
	AuthProjectDomain string
}

type HelmConfig struct {
	ChartDirectory string
}

type KubernikusConfig struct {
	Domain      string
	Namespace   string
	ProjectID   string
	NetworkID   string
	KubeConfig  string
	KubeContext string
	Controllers map[string]Controller
}

type Config struct {
	Openstack  OpenstackConfig
	Kubernikus KubernikusConfig
	Helm       HelmConfig
	Images     version.ImageRegistry
}

type Clients struct {
	Kubernikus kubernikus_clientset.Interface
	Kubernetes kubernetes_clientset.Interface
	Satellites kubernetes.SharedClientFactory

	Helm *helm.Client
}

func (c *Config) GetHelm3Config(releaseNamespace string, logger log.Logger) (*action.Configuration, error) {
	config := &action.Configuration{}
	err := config.Init(kube.GetConfig(c.Kubernikus.KubeConfig, c.Kubernikus.KubeContext, releaseNamespace), releaseNamespace, "secrets", func(format string, v ...interface{}) {
		logger.Log("component", "helm3", "msg", fmt.Sprintf(format, v))
	})
	if err != nil {
		return nil, err
	}
	return config, nil
}

type Factories struct {
	Openstack  openstack.SharedOpenstackClientFactory
	Kubernikus kubernikus_informers.SharedInformerFactory
	Kubernetes kubernetes_informers.SharedInformerFactory

	NodesObservatory *nodeobservatory.InformerFactory
}
