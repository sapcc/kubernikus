package ground

import (
	"fmt"

	rbac "k8s.io/api/rbac/v1beta1"
	storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/ground/bootstrap/dns"
)

func SeedKluster(client clientset.Interface, kluster *v1.Kluster) error {
	if err := SeedAllowBootstrapTokensToPostCSRs(client); err != nil {
		return err
	}
	if err := SeedAutoApproveNodeBootstrapTokens(client); err != nil {
		return err
	}
	if err := SeedKubernikusAdmin(client); err != nil {
		return err
	}
	if err := SeedKubernikusMember(client); err != nil {
		return err
	}
	if err := SeedCinderStorageClass(client); err != nil {
		return err
	}
	if err := SeedAllowCertificateControllerToDeleteCSRs(client); err != nil {
		return err
	}
	if err := dns.SeedKubeDNS(client, "", "", kluster.Spec.DNSDomain, kluster.Spec.DNSAddress); err != nil {
		return err
	}
	return nil

}

func SeedCinderStorageClass(client clientset.Interface) error {
	storageClass := storage.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cinder-default",
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

func SeedKubernikusMember(client clientset.Interface) error {
	return CreateOrUpdateRoleBinding(client, &rbac.RoleBinding{
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

// addresses https://github.com/kubernetes/kubernetes/issues/59351
func SeedAllowCertificateControllerToDeleteCSRs(client clientset.Interface) error {
	return CreateOrUpdateClusterRole(client, &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system:controller:certificate-controller",
			Annotations: map[string]string{
				"rbac.authorization.kubernetes.io/autoupdate": "true",
			},
			Labels: map[string]string{
				"kubernetes.io/bootstrapping": "rbac-defaults",
			},
		},
		Rules: []rbac.PolicyRule{
			rbac.PolicyRule{
				Verbs:     []string{"delete", "get", "list", "watch"},
				APIGroups: []string{"certificates.k8s.io"},
				Resources: []string{"certificatesigningrequests"},
			},
			rbac.PolicyRule{
				Verbs:     []string{"update"},
				APIGroups: []string{"certificates.k8s.io"},
				Resources: []string{"certificatesigningrequests/approval", "certificatesigningrequests/status"},
			},
			rbac.PolicyRule{
				Verbs:     []string{"create"},
				APIGroups: []string{"authorization.k8s.io"},
				Resources: []string{"subjectaccessreviews"},
			},
			rbac.PolicyRule{
				Verbs:     []string{"create", "patch", "update"},
				APIGroups: []string{""}, //looks funny but is in the default rule ...
				Resources: []string{"events"},
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
			rbac.PolicyRule{
				Verbs:     []string{"create"},
				APIGroups: []string{"certificates.k8s.io"},
				Resources: []string{"certificatesigningrequests/nodeclient"},
			},
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

func CreateOrUpdateRoleBinding(client clientset.Interface, roleBinding *rbac.RoleBinding) error {
	if _, err := client.RbacV1beta1().RoleBindings(roleBinding.Namespace).Create(roleBinding); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create RBAC rolebinding: %v", err)
		}

		if _, err := client.RbacV1beta1().RoleBindings(roleBinding.Namespace).Update(roleBinding); err != nil {
			return fmt.Errorf("unable to update RBAC rolebinding: %v", err)
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
