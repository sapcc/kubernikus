package csi

import (
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/pkg/errors"
	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/version"
)

type images struct {
	ImageAttacher      string
	ImageProvisioner   string
	ImageSnapshotter   string
	ImageResizer       string
	ImageLivenessProbe string
	ImageCSIPlugin     string
	ImageNodeDriver    string
}

func SeedCinderCSIPlugin(client clientset.Interface, dynamicClient dynamic.Interface, klusterSecret *kubernikus_v1.Secret, versions version.KlusterVersion) error {
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-config",
			Namespace: "kube-system",
		},
		StringData: map[string]string{
			"cloudprovider.conf": `[Global]
auth-url="https://identity-3.` + klusterSecret.Openstack.Region + `.cloud.sap/v3/"
domain-name="kubernikus"
tenant-id="` + klusterSecret.Openstack.ProjectID + `"
username="` + klusterSecret.Openstack.Username + `"
password="` + klusterSecret.Openstack.Password + `"
region="` + klusterSecret.Openstack.Region + `"

[BlockStorage]
`,
		},
	}

	err := createSecret(client, &secret)
	if err != nil {
		return errors.Wrap(err, "CSISecret")
	}

	err = createServiceAccount(client, CSIServiceAccountController)
	if err != nil {
		return errors.Wrap(err, "CSIServiceAccountController")
	}

	err = createServiceAccount(client, CSIServiceAccountNode)
	if err != nil {
		return errors.Wrap(err, "CSIServiceAccountNode")
	}

	err = createRole(client, CSIRole)
	if err != nil {
		return errors.Wrap(err, "CSIRole")
	}

	err = createClusterRole(client, CSIClusterRoleAttacher)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleAttacher")
	}

	err = createClusterRole(client, CSIClusterRoleNodePlugin)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleNodePlugin")
	}

	err = createClusterRole(client, CSIClusterRoleProvisioner)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleProvisioner")
	}

	err = createClusterRole(client, CSIClusterRoleResizer)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleResizer")
	}

	err = createClusterRole(client, CSIClusterRoleSnapshotter)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleSnapshotter")
	}

	err = createRoleBinding(client, CSIRoleBindingResizer)
	if err != nil {
		return errors.Wrap(err, "CSIRoleBindingResizer")
	}

	err = createClusterRoleBinding(client, CSIClusterRoleBindingAttacher)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingAttacher")
	}

	err = createClusterRoleBinding(client, CSIClusterRoleBindingNodePlugin)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingNodePlugin")
	}

	err = createClusterRoleBinding(client, CSIClusterRoleBindingProvisioner)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingProvisioner")
	}

	err = createClusterRoleBinding(client, CSIClusterRoleBindingResizer)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingResizer")
	}

	err = createClusterRoleBinding(client, CSIClusterRoleBindingSnapshotter)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingSnapshotter")
	}

	/*
		err = createService(client, CSIControllerService)
		if err != nil {
			return errors.Wrap(err, "CSIControllerService")
		}
	*/

	data := images{
		versions.CSIAttacher.Repository + ":" + versions.CSIAttacher.Tag,
		versions.CSIProvisioner.Repository + ":" + versions.CSIProvisioner.Tag,
		versions.CSISnapshotter.Repository + ":" + versions.CSISnapshotter.Tag,
		versions.CSIResizer.Repository + ":" + versions.CSIResizer.Tag,
		versions.CSILivenessprobe.Repository + ":" + versions.CSILivenessprobe.Tag,
		versions.CinderCSIPlugin.Repository + ":" + versions.CinderCSIPlugin.Tag,
		versions.CSINodeDriver.Repository + ":" + versions.CSINodeDriver.Tag,
	}

	err = createStatefulSet(client, CSIStatefulSet, data)
	if err != nil {
		return errors.Wrap(err, "CSIStatefulSet")
	}

	err = createDaemonSet(client, CSIDaemonsSet, data)
	if err != nil {
		return errors.Wrap(err, "CSIDaemonsSet")
	}

	err = createCSIDriver(dynamicClient, CSIDriver)
	if err != nil {
		return errors.Wrap(err, "CSIDriver")
	}

	return nil
}

func createCSIDriver(dynamicClient dynamic.Interface, manifest string) error {
	csiDriverRes := schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers"}

	var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	csiDriver := &unstructured.Unstructured{}
	_, _, err := decUnstructured.Decode([]byte(manifest), nil, csiDriver)
	if err != nil {
		return errors.Wrap(err, "Decode")
	}

	_, err = dynamicClient.Resource(csiDriverRes).Create(csiDriver, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "Create")
	}

	return nil
}

func createSecret(client clientset.Interface, secret *v1.Secret) error {
	if err := bootstrap.CreateOrUpdateSecret(client, secret); err != nil {
		return err
	}

	return nil
}

func createStatefulSet(client clientset.Interface, manifest string, data images) error {
	template, err := bootstrap.RenderManifest(manifest, data)
	if err != nil {
		return err
	}

	statefulSet, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &apps.StatefulSet{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateStatefulSet(client, statefulSet.(*apps.StatefulSet)); err != nil {
		return err
	}

	return nil
}

func createService(client clientset.Interface, manifest string) error {
	template, err := bootstrap.RenderManifest(manifest, nil)
	if err != nil {
		return err
	}

	service, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &v1.Service{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateService(client, service.(*v1.Service)); err != nil {
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

func createRoleBinding(client clientset.Interface, manifest string) error {
	template, err := bootstrap.RenderManifest(manifest, nil)
	if err != nil {
		return err
	}

	roleBinding, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &rbac.RoleBinding{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateRoleBindingV1(client, roleBinding.(*rbac.RoleBinding)); err != nil {
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

func createRole(client clientset.Interface, manifest string) error {
	template, err := bootstrap.RenderManifest(manifest, nil)
	if err != nil {
		return err
	}

	role, _, err := serializer.NewCodecFactory(clientsetscheme.Scheme).UniversalDeserializer().Decode(template, nil, &rbac.Role{})
	if err != nil {
		return err
	}

	if err := bootstrap.CreateOrUpdateRole(client, role.(*rbac.Role)); err != nil {
		return err
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
