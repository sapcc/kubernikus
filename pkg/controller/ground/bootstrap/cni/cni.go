package cni

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/version"
)

type images struct {
	CNIPlugins       string
	Flannel          string
	FlannelCNIPlugin string
}

func SeedCNIConfig(client clientset.Interface, clusterCIDR string, versions version.KlusterVersion) error {

	if err := createServiceAccount(client, CNIServiceAccount); err != nil {
		return fmt.Errorf("Failed to bootstrap cni service account: %w", err)
	}
	if err := createClusterRole(client, CNIClusterRole); err != nil {
		return fmt.Errorf("Failed to bootstrap cni cluster role: %w", err)
	}
	if err := createClusterRoleBinding(client, CNIClusterRoleBinding); err != nil {
		return fmt.Errorf("Failed to bootstrap cni cluster role binding: %w", err)
	}
	img := images{
		CNIPlugins:       versions.CNIPlugins.Repository + ":" + versions.CNIPlugins.Tag,
		Flannel:          versions.Flannel.Repository + ":" + versions.Flannel.Tag,
		FlannelCNIPlugin: versions.FlannelCNIPlugin.Repository + ":" + versions.FlannelCNIPlugin.Tag,
	}

	if err := createCNIConfigMap(client, CNIConfigMap, clusterCIDR); err != nil {
		return fmt.Errorf("Failed to bootstrap cni cluster role binding: %w", err)
	}

	if err := createDaemonSet(client, CNIDaemonSet, img); err != nil {
		return fmt.Errorf("Failed to bootstrap cni cluster role binding: %w", err)
	}

	return nil

}

func createServiceAccount(client clientset.Interface, manifest string) error {
	template, err := bootstrap.RenderManifest(manifest, nil)
	if err != nil {
		return err
	}

	serviceAccount, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &v1.ServiceAccount{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateServiceAccount(client, serviceAccount.(*v1.ServiceAccount)); err != nil {
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

func createCNIConfigMap(client clientset.Interface, manifest string, clusterCIDR string) error {

	data := struct{ ClusterCIDR string }{clusterCIDR}

	template, err := bootstrap.RenderManifest(manifest, data)
	if err != nil {
		return err
	}
	configMap, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &v1.ConfigMap{})
	if err != nil {
		return err
	}

	return bootstrap.CreateOrUpdateConfigMap(client, configMap.(*v1.ConfigMap))
}

func createDaemonSet(client clientset.Interface, manifest string, data images) error {
	template, err := bootstrap.RenderManifest(manifest, data)
	if err != nil {
		return err
	}

	daemonset, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &apps.DaemonSet{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateDaemonset(client, daemonset.(*apps.DaemonSet)); err != nil {
		return err
	}

	return nil
}
