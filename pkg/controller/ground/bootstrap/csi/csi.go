package csi

import (
	"github.com/pkg/errors"
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

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/version"
)

type images struct {
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

	err = bootstrap.CreateServiceAccountFromTemplate(client, CSIServiceAccountController, nil)
	if err != nil {
		return errors.Wrap(err, "CSIServiceAccountController")
	}

	err = bootstrap.CreateServiceAccountFromTemplate(client, CSIServiceAccountNode, nil)
	if err != nil {
		return errors.Wrap(err, "CSIServiceAccountNode")
	}

	err = createRole(client, CSIRole)
	if err != nil {
		return errors.Wrap(err, "CSIRole")
	}

	err = bootstrap.CreateClusterRoleFromTemplate(client, CSIClusterRoleAttacher, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleAttacher")
	}

	err = bootstrap.CreateClusterRoleFromTemplate(client, CSIClusterRoleNodePlugin, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleNodePlugin")
	}

	err = bootstrap.CreateClusterRoleFromTemplate(client, CSIClusterRoleProvisioner, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleProvisioner")
	}

	err = bootstrap.CreateClusterRoleFromTemplate(client, CSIClusterRoleResizer, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleResizer")
	}

	err = bootstrap.CreateClusterRoleFromTemplate(client, CSIClusterRoleSnapshotter, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleSnapshotter")
	}

	err = createRoleBinding(client, CSIRoleBindingResizer)
	if err != nil {
		return errors.Wrap(err, "CSIRoleBindingResizer")
	}

	err = bootstrap.CreateClusterRoleBindingFromTemplate(client, CSIClusterRoleBindingAttacher, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingAttacher")
	}

	err = bootstrap.CreateClusterRoleBindingFromTemplate(client, CSIClusterRoleBindingNodePlugin, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingNodePlugin")
	}

	err = bootstrap.CreateClusterRoleBindingFromTemplate(client, CSIClusterRoleBindingProvisioner, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingProvisioner")
	}

	err = bootstrap.CreateClusterRoleBindingFromTemplate(client, CSIClusterRoleBindingResizer, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingResizer")
	}

	err = bootstrap.CreateClusterRoleBindingFromTemplate(client, CSIClusterRoleBindingSnapshotter, nil)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleBindingSnapshotter")
	}

	data := images{
		versions.CSILivenessProbe.Repository + ":" + versions.CSILivenessProbe.Tag,
		versions.CinderCSIPlugin.Repository + ":" + versions.CinderCSIPlugin.Tag,
		versions.CSINodeDriver.Repository + ":" + versions.CSINodeDriver.Tag,
	}

	err = bootstrap.CreateDaemonSetFromTemplate(client, CSIDaemonsSet, data)
	if err != nil {
		return errors.Wrap(err, "CSIDaemonsSet")
	}

	gvrCSIDriver := schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers"}
	err = createDynamicResource(dynamicClient, CSIDriver, gvrCSIDriver)
	if err != nil {
		return errors.Wrap(err, "CSIDriver")
	}

	gvrSnapClassCRD := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	err = createDynamicResource(dynamicClient, CSISnapshotCRDVolumeSnapshotClass, gvrSnapClassCRD)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotCRDVolumeSnapshotClass")
	}

	gvrSnapContentCRD := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	err = createDynamicResource(dynamicClient, CSISnapshotCRDVolumeSnapshotContent, gvrSnapContentCRD)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotCRDVolumeSnapshotContent")
	}

	gvrSnapCRD := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	err = createDynamicResource(dynamicClient, CSISnapshotCRDVolumeSnapshot, gvrSnapCRD)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotCRDVolumeSnapshot")
	}

	err = bootstrap.CreateClusterRoleFromTemplate(client, CSISnapshotControllerClusterRole, nil)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotControllerClusterRole")
	}

	err = bootstrap.CreateClusterRoleBindingFromTemplate(client, CSISnapshotControllerClusterRoleBinding, nil)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotControllerClusterRoleBinding")
	}

	err = createRole(client, CSISnapshotControllerRole)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotControllerRole")
	}

	err = createRoleBinding(client, CSISnapshotControllerRoleBinding)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotControllerRoleBinding")
	}

	gvrSnapClass := schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1beta1", Resource: "volumesnapshotclasses"}
	err = createDynamicResource(dynamicClient, CSIVolumeSnapshotClass, gvrSnapClass)
	if err != nil {
		return errors.Wrap(err, "CSIVolumeSnapshotClass")
	}

	return nil
}

func createDynamicResource(dynamicClient dynamic.Interface, manifest string, gvr schema.GroupVersionResource) error {
	var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	resource := &unstructured.Unstructured{}
	_, _, err := decUnstructured.Decode([]byte(manifest), nil, resource)
	if err != nil {
		return errors.Wrap(err, "Decode")
	}

	_, err = dynamicClient.Resource(gvr).Create(resource, metav1.CreateOptions{})
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
