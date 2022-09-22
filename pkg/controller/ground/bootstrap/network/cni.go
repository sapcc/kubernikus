package network

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

type cni struct {
	CNIPlugins       string
	Flannel          string
	FlannelCNIPlugin string
}

type wormhole struct {
	Wormhole string
	Listen   string
}

type kubeProxy struct {
	KubeProxy string
}

func SeedNetwork(client clientset.Interface, versions version.KlusterVersion, clusterCIDR, apiserverURL, apiserverIP string, apiserverPort int64) error {

	if err := SeedCNIConfig(client, versions, clusterCIDR, apiserverURL); err != nil {
		return fmt.Errorf("cni config: %w", err)
	}

	if err := SeedKubeProxy(client, versions, clusterCIDR, apiserverURL); err != nil {
		return fmt.Errorf("kube-proxy: %w", err)
	}
	if err := SeedWormhole(client, versions, apiserverIP, apiserverPort); err != nil {
		return fmt.Errorf("wormhole: %w", err)
	}
	return nil

}

func SeedKubeProxy(client clientset.Interface, versions version.KlusterVersion, clusterCIDR, apiServerURL string) error {
	if err := createServiceAccount(client, KubeProxyServiceAccount); err != nil {
		return fmt.Errorf("Failed to bootstrap kube-proxy service account: %w", err)
	}
	if err := createClusterRoleBinding(client, KubeProxyClusterRoleBinding); err != nil {
		return fmt.Errorf("Failed to bootstrap kube-proxy cluster role binding: %w", err)
	}
	cmData := struct {
		ClusterCIDR  string
		APIServerURL string
	}{clusterCIDR, apiServerURL}
	if err := createConfigMap(client, KubeProxyConfigMap, cmData); err != nil {
		return fmt.Errorf("Failed to bootstrap kube-proxy config map: %w", err)
	}
	data := kubeProxy{
		KubeProxy: versions.KubeProxy.Repository + ":" + versions.KubeProxy.Tag,
	}
	if err := createDaemonSet(client, KubeProxyDaemonSet, data); err != nil {
		return fmt.Errorf("Failed to bootstrap kube-proxy daemon set: %w", err)
	}
	return nil
}

func SeedWormhole(client clientset.Interface, versions version.KlusterVersion, apiServerIP string, apiServerPort int64) error {
	data := wormhole{
		Wormhole: versions.Wormhole.Repository + ":" + version.GitCommit,
		Listen:   fmt.Sprintf("%s:%d", apiServerIP, apiServerPort),
	}
	if err := createDaemonSet(client, WormholeDaemonSet, data); err != nil {
		return fmt.Errorf("Failed to bootstrap wormhole daemon set: %w", err)
	}
	return nil
}

func SeedCNIConfig(client clientset.Interface, versions version.KlusterVersion, clusterCIDR, apiserverURL string) error {

	if err := createServiceAccount(client, CNIServiceAccount); err != nil {
		return fmt.Errorf("Failed to bootstrap cni service account: %w", err)
	}
	if err := createClusterRole(client, CNIClusterRole); err != nil {
		return fmt.Errorf("Failed to bootstrap cni cluster role: %w", err)
	}
	if err := createClusterRoleBinding(client, CNIClusterRoleBinding); err != nil {
		return fmt.Errorf("Failed to bootstrap cni cluster role binding: %w", err)
	}

	cmData := struct {
		ClusterCIDR  string
		APIServerURL string
	}{clusterCIDR, apiserverURL}
	if err := createConfigMap(client, CNIConfigMap, cmData); err != nil {
		return fmt.Errorf("Failed to bootstrap cni config map: %w", err)
	}

	dsVals := cni{
		CNIPlugins:       versions.CNIPlugins.Repository + ":" + versions.CNIPlugins.Tag,
		Flannel:          versions.Flannel.Repository + ":" + versions.Flannel.Tag,
		FlannelCNIPlugin: versions.FlannelCNIPlugin.Repository + ":" + versions.FlannelCNIPlugin.Tag,
	}

	if err := createDaemonSet(client, CNIDaemonSet, dsVals); err != nil {
		return fmt.Errorf("Failed to bootstrap cni daemon set: %w", err)
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

func createConfigMap(client clientset.Interface, manifest string, data interface{}) error {

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

func createDaemonSet(client clientset.Interface, manifest string, data interface{}) error {
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
