package ccm

import (
	"github.com/pkg/errors"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
)

func SeedCloudControllerManagerRoles(client clientset.Interface) error {
	err := createClusterRole(client, CCMClusterRole)
	if err != nil {
		return errors.Wrap(err, "CCMClusterRole")
	}

	err = createClusterRole(client, CCMClusterRoleNode)
	if err != nil {
		return errors.Wrap(err, "CCMClusterRoleNode")
	}

	err = createClusterRoleBinding(client, CCMClusterRoleBinding)
	if err != nil {
		return errors.Wrap(err, "CCMClusterRoleBinding")
	}

	err = createClusterRoleBinding(client, CCMClusterRoleBindingNode)
	if err != nil {
		return errors.Wrap(err, "CCMClusterRoleBindingNode")
	}

	return nil
}

func createClusterRoleBinding(client clientset.Interface, manifest string) error {
	template, err := bootstrap.RenderManifest(manifest, nil)
	if err != nil {
		return err
	}

	clusterRoleBinding, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &rbac.ClusterRoleBinding{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateClusterRoleBindingV1(client, clusterRoleBinding.(*rbac.ClusterRoleBinding)); err != nil {
		return err
	}

	return nil
}

func createClusterRole(client clientset.Interface, manifest string) error {
	template, err := bootstrap.RenderManifest(manifest, nil)
	if err != nil {
		return err
	}

	clusterRole, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &rbac.ClusterRole{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateClusterRoleV1(client, clusterRole.(*rbac.ClusterRole)); err != nil {
		return err
	}

	return nil
}
