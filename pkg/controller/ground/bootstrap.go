package ground

import (
	"fmt"

	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/dns"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
	storage "k8s.io/client-go/pkg/apis/storage/v1"
)

func SeedKluster(client clientset.Interface) error {
	if err := SeedAllowBootstrapTokensToPostCSRs(client); err != nil {
		return err
	}
	if err := SeedAutoApproveNodeBootstrapTokens(client); err != nil {
		return err
	}
	if err := SeedKubernikusAdmin(client); err != nil {
		return err
	}
	if err := SeedCinderStorageClass(client); err != nil {
		return err
	}
	if err := dns.SeedKubeDNS(client, "", "", "", ""); err != nil {
		return err
	}
	return nil

}

func SeedCinderStorageClass(client clientset.Interface) error {
	storageClass := storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cinder:default",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		Provisioner: "kubernetes.io/cinder",
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
	return CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
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

func SeedAllowBootstrapTokensToPostCSRs(client clientset.Interface) error {
	return CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
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

func SeedAutoApproveNodeBootstrapTokens(client clientset.Interface) error {
	err := CreateOrUpdateClusterRole(client, &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubernikus:approve-node-client-csr",
		},
		Rules: []rbac.PolicyRule{
			rbac.NewRule("create").Groups("certificates.k8s.io").Resources("certificatesigningrequests/nodeclient").RuleOrDie(),
		},
	})
	if err != nil {
		return err
	}

	return CreateOrUpdateClusterRoleBinding(client, &rbac.ClusterRoleBinding{
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

func CreateOrUpdateClusterRoleBinding(client clientset.Interface, clusterRoleBinding *rbac.ClusterRoleBinding) error {
	if _, err := client.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create RBAC clusterrolebinding: %v", err)
		}

		if _, err := client.RbacV1beta1().ClusterRoleBindings().Update(clusterRoleBinding); err != nil {
			return fmt.Errorf("unable to update RBAC clusterrolebinding: %v", err)
		}
	}
	return nil
}

func CreateOrUpdateClusterRole(client clientset.Interface, clusterRole *rbac.ClusterRole) error {
	if _, err := client.RbacV1beta1().ClusterRoles().Create(clusterRole); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create RBAC clusterrole: %v", err)
		}

		if _, err := client.RbacV1beta1().ClusterRoles().Update(clusterRole); err != nil {
			return fmt.Errorf("unable to update RBAC clusterrole: %v", err)
		}
	}
	return nil
}
