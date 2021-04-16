package kubeproxy

import (
	"net/url"

	clientset "k8s.io/client-go/kubernetes"

	kubernikus "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/version"
)

type data struct {
	ClusterCIDR   *string
	Image         string
	ApiserverHost string
}

func SeedKubeProxy(client clientset.Interface, versions version.KlusterVersion, kluster *kubernikus.Kluster) error {
	url, err := url.Parse(kluster.Status.Apiserver)
	if err != nil {
		return err
	}

	vars := data{
		kluster.Spec.ClusterCIDR,
		versions.KubeProxy.Repository + ":" + versions.KubeProxy.Tag,
		url.Host,
	}

	if err := bootstrap.CreateServiceAccountFromTemplate(client, KubeProxyServiceAccount, nil); err != nil {
		return err
	}

	if err := bootstrap.CreateClusterRoleBindingFromTemplate(client, KubeProxyClusterRoleBinding, nil); err != nil {
		return err
	}

	if err := bootstrap.CreateConfigMapFromTemplate(client, KubeProxyConfigmap, vars); err != nil {
		return err
	}

	if err := bootstrap.CreateDaemonSetFromTemplate(client, KubeProxyDaemonset, vars); err != nil {
		return err
	}

	return nil
}
