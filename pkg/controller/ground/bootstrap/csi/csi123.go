package csi

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/version"
)

func SeedCinderCSIPlugin123(client clientset.Interface, dynamicClient dynamic.Interface, klusterSecret *kubernikus_v1.Secret, versions version.KlusterVersion) error {
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
rescan-on-resize = yes
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

	err = createClusterRole(client, CSIClusterRoleAttacher123)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleAttacher")
	}

	err = createClusterRole(client, CSIClusterRoleNodePlugin)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleNodePlugin")
	}

	err = createClusterRole(client, CSIClusterRoleProvisioner123)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleProvisioner")
	}

	err = createClusterRole(client, CSIClusterRoleResizer123)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleResizer")
	}

	err = createClusterRole(client, CSIClusterRoleSnapshotter123)
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

	data := images{
		versions.CSILivenessProbe.Repository + ":" + versions.CSILivenessProbe.Tag,
		versions.CinderCSIPlugin.Repository + ":" + versions.CinderCSIPlugin.Tag,
		versions.CSINodeDriver.Repository + ":" + versions.CSINodeDriver.Tag,
	}

	err = createDaemonSet(client, CSIDaemonsSet, data)
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

	err = createClusterRole(client, CSISnapshotControllerClusterRole)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotControllerClusterRole")
	}

	err = createClusterRoleBinding(client, CSISnapshotControllerClusterRoleBinding)
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

	gvrSnapClass := schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotclasses"}
	err = createDynamicResource(dynamicClient, CSIVolumeSnapshotClass, gvrSnapClass)
	if err != nil {
		return errors.Wrap(err, "CSIVolumeSnapshotClass")
	}

	return nil
}

func SeedCinderCSIRoles123(client clientset.Interface) error {
	err := createRole(client, CSIRole)
	if err != nil {
		return errors.Wrap(err, "CSIRole")
	}

	err = createClusterRole(client, CSISnapshotControllerClusterRole)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotControllerClusterRole")
	}

	err = createClusterRole(client, CSIClusterRoleAttacher123)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleAttacher")
	}

	err = createClusterRole(client, CSIClusterRoleNodePlugin)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleNodePlugin")
	}

	err = createClusterRole(client, CSIClusterRoleProvisioner123)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleProvisioner")
	}

	err = createClusterRole(client, CSIClusterRoleResizer123)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleResizer")
	}

	err = createClusterRole(client, CSIClusterRoleSnapshotter123)
	if err != nil {
		return errors.Wrap(err, "CSIClusterRoleSnapshotter")
	}

	err = createRole(client, CSISnapshotControllerRole)
	if err != nil {
		return errors.Wrap(err, "CSISnapshotControllerRole")
	}

	return nil
}
