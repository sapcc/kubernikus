package ground

import (
	"fmt"

	"github.com/pkg/errors"
	rbac "k8s.io/api/rbac/v1beta1"
	storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	openstack_project "github.com/sapcc/kubernikus/pkg/client/openstack/project"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/csi"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/dns"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/gpu"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/version"
)

func SeedKluster(clients config.Clients, factories config.Factories, images version.ImageRegistry, kluster *v1.Kluster) error {
	kubernetes, err := clients.Satellites.ClientFor(kluster)
	if err != nil {
		return err
	}

	if err := SeedAllowBootstrapTokensToPostCSRs(kubernetes); err != nil {
		return errors.Wrap(err, "seed allow bootstrap tokens to post CSRs")
	}
	if err := SeedAutoApproveNodeBootstrapTokens(kubernetes); err != nil {
		return errors.Wrap(err, "seed auto approve node bootstrap tokens")
	}
	if err := SeedAutoRenewalNodeCertificates(kubernetes); err != nil {
		return errors.Wrap(err, "seed auto renewal node certificates")
	}
	if err := SeedKubernikusAdmin(kubernetes); err != nil {
		return errors.Wrap(err, "seed kubernikus admin")
	}
	if err := SeedKubernikusMember(kubernetes); err != nil {
		return errors.Wrap(err, "seed kubernikus member")
	}
	if !kluster.Spec.NoCloud {
		openstack, err := factories.Openstack.ProjectAdminClientFor(kluster.Account())
		if err != nil {
			return err
		}
		useCSI, _ := util.KlusterVersionConstraint(kluster, ">= 1.20")
		if err := SeedCinderStorageClasses(kubernetes, openstack, useCSI); err != nil {
			return errors.Wrap(err, "seed cinder storage classes")
		}
	}
	if err := SeedAllowApiserverToAccessKubeletAPI(kubernetes); err != nil {
		return errors.Wrap(err, "seed allow apiserver access to kubelet api")
	}
	coreDNSImage := ""
	if images.Versions[kluster.Spec.Version].CoreDNS.Repository != "" &&
		images.Versions[kluster.Spec.Version].CoreDNS.Tag != "" {
		coreDNSImage = images.Versions[kluster.Spec.Version].CoreDNS.Repository + ":" + images.Versions[kluster.Spec.Version].CoreDNS.Tag
	}
	if ok, _ := util.KlusterVersionConstraint(kluster, ">= 1.16"); ok {
		if err := dns.SeedCoreDNS116(kubernetes, coreDNSImage, kluster.Spec.DNSDomain, kluster.Spec.DNSAddress); err != nil {
			return errors.Wrap(err, "seed coredns")
		}
	} else if ok, _ := util.KlusterVersionConstraint(kluster, ">= 1.13"); ok {
		if err := dns.SeedCoreDNS(kubernetes, coreDNSImage, kluster.Spec.DNSDomain, kluster.Spec.DNSAddress); err != nil {
			return errors.Wrap(err, "seed coredns")
		}
	} else {
		if err := dns.SeedKubeDNS(kubernetes, "", "", kluster.Spec.DNSDomain, kluster.Spec.DNSAddress); err != nil {
			return errors.Wrap(err, "seed kubedns")
		}
	}

	if ok, _ := util.KlusterVersionConstraint(kluster, ">= 1.10"); ok {
		if err := gpu.SeedGPUSupport(kubernetes); err != nil {
			return errors.Wrap(err, "seed GPU support")
		}
	}

	if ok, _ := util.KlusterVersionConstraint(kluster, ">= 1.20"); ok {
		dynamicKubernetes, err := clients.Satellites.DynamicClientFor(kluster)
		if err != nil {
			return errors.Wrap(err, "dynamic client")
		}

		klusterSecret, err := util.KlusterSecret(clients.Kubernetes, kluster)
		if err != nil {
			return errors.Wrap(err, "get kluster secret")
		}

		if err := csi.SeedCinderCSIPlugin(kubernetes, dynamicKubernetes, klusterSecret, images.Versions[kluster.Spec.Version]); err != nil {
			return errors.Wrap(err, "seed cinder CSI plugin")
		}
	}

	if err := SeedOpenStackClusterRoleBindings(kubernetes); err != nil {
		return errors.Wrap(err, "seed openstack cluster role bindings")
	}

	return nil
}

func SeedCinderStorageClasses(client clientset.Interface, openstack openstack_project.ProjectClient, useCSI bool) error {
	if err := createStorageClass(client, "cinder-default", "", true, useCSI); err != nil {
		return err
	}

	metadata, err := openstack.GetMetadata()
	if err != nil {
		return err
	}

	for _, avz := range metadata.AvailabilityZones {
		name := fmt.Sprintf("cinder-zone-%s", avz.Name[len(avz.Name)-1:])
		if err := createStorageClass(client, name, avz.Name, false, useCSI); err != nil {
			return err
		}
	}

	return nil
}

func createStorageClass(client clientset.Interface, name, avz string, isDefault bool, useCSI bool) error {
	provisioner := "kubernetes.io/cinder"
	expansion := false

	if useCSI {
		provisioner = "cinder.csi.openstack.org"
		expansion = true
	}

	mode := storage.VolumeBindingImmediate

	if avz == "" {
		mode = storage.VolumeBindingWaitForFirstConsumer
	}

	storageClass := storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Provisioner:          provisioner,
		VolumeBindingMode:    &mode,
		AllowVolumeExpansion: &expansion,
	}

	if isDefault {
		storageClass.Annotations = map[string]string{
			"storageclass.kubernetes.io/is-default-class": "true",
		}
	}

	if avz != "" {
		storageClass.Parameters = map[string]string{
			"availability": avz,
		}
	}

	if _, err := client.StorageV1().StorageClasses().Create(&storageClass); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create storage class: %v", err)
		}

		if _, err := client.StorageV1().StorageClasses().Update(&storageClass); err != nil {
			return fmt.Errorf("unable to update storage class: %v", err)
		}
	}

	return nil
}

func SeedKubernikusAdmin(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:admin",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbac.Subject{
			{
				Kind: rbac.GroupKind,
				Name: "os:kubernetes_admin",
			},
		},
	})
}

func SeedKubernikusMember(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateRoleBinding(client, &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernikus:member",
			Namespace: "default",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "Role",
			Name:     "edit",
		},
		Subjects: []rbac.Subject{
			{
				Kind: rbac.GroupKind,
				Name: "os:kubernetes_member",
			},
		},
	})
}

func SeedAllowBootstrapTokensToPostCSRs(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:kubelet-bootstrap",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "system:node-bootstrapper",
		},
		Subjects: []rbac.Subject{
			{
				Kind: rbac.GroupKind,
				Name: "system:bootstrappers",
			},
		},
	})
}

func SeedAllowApiserverToAccessKubeletAPI(client clientset.Interface) error {
	return bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:apiserver-kubeletapi",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "system:kubelet-api-admin",
		},
		Subjects: []rbac.Subject{
			{
				Kind: rbac.UserKind,
				Name: "apiserver",
			},
		},
	})
}

func SeedAutoApproveNodeBootstrapTokens(client clientset.Interface) error {
	err := bootstrap.CreateOrUpdateClusterRole(client, &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:approve-node-client-csr",
		},
		Rules: []rbac.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"certificates.k8s.io"},
				Resources: []string{"certificatesigningrequests/nodeclient"},
			},
		},
	})
	if err != nil {
		return err
	}

	return bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:node-client-csr-autoapprove",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "kubernikus:approve-node-client-csr",
		},
		Subjects: []rbac.Subject{
			{
				Kind: "Group",
				Name: "system:bootstrappers",
			},
		},
	})
}

func SeedAutoRenewalNodeCertificates(client clientset.Interface) error {
	err := bootstrap.CreateOrUpdateClusterRole(client, &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient",
		},
		Rules: []rbac.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"certificates.k8s.io"},
				Resources: []string{"certificatesigningrequests/selfnodeclient"},
			},
		},
	})
	if err != nil {
		return err
	}

	return bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:auto-approve-renewals-for-nodes",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient",
		},
		Subjects: []rbac.Subject{
			{
				APIGroup: rbac.GroupName,
				Kind:     "Group",
				Name:     "system:nodes",
			},
		},
	})
}

func SeedOpenStackClusterRoleBindings(client clientset.Interface) error {

	err := bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:openstack-kubernetes-admin",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbac.Subject{
			{
				Kind: "Group",
				Name: "openstack_role:kubernetes_admin",
			},
			{
				Kind: "User",
				// It is the marshall & b64enc of the protobuf message IDTokenSubject: https://github.com/dexidp/dex/blob/master/server/oauth2.go#L300
				// User ID: 00000000-0000-0000-0000-000000000001 ConnID: local
				Name: "CiQwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDESBWxvY2Fs",
				// For claims, we are using "sub" instead of "email" since some technical users missing emails
				// If we switch to email, we can directly use email as Name field above
			},
		},
	})

	if err != nil {
		return err
	}

	err = bootstrap.CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:openstack-kubernetes-member",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "view",
		},
		Subjects: []rbac.Subject{
			{
				Kind: "Group",
				Name: "openstack_role:kubernetes_member",
			},
		},
	})

	if err != nil {
		return err
	}

	return nil
}
