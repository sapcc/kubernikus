package flannel

import (
	clientset "k8s.io/client-go/kubernetes"

	kubernikus "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/version"
)

type data struct {
	ClusterCIDR *string
	Image       string
}

func SeedFlannel(client clientset.Interface, versions version.KlusterVersion, kluster *kubernikus.Kluster) error {
	vars := data{
		kluster.Spec.ClusterCIDR,
		versions.Flannel.Repository + ":" + versions.Flannel.Tag,
	}

	if err := bootstrap.CreateServiceAccountFromTemplate(client, FlannelServiceAccount, nil); err != nil {
		return err
	}

	if err := bootstrap.CreateClusterRoleFromTemplate(client, FlannelClusterRole, nil); err != nil {
		return err
	}

	if err := bootstrap.CreateClusterRoleBindingFromTemplate(client, FlannelClusterRoleBinding, nil); err != nil {
		return err
	}

	if err := bootstrap.CreateConfigMapFromTemplate(client, FlannelConfigmap, vars); err != nil {
		return err
	}

	if err := bootstrap.CreateDaemonSetFromTemplate(client, FlannelDaemonset, vars); err != nil {
		return err
	}

	return nil
}
