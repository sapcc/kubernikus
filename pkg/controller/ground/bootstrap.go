package ground

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	rbac "k8s.io/client-go/pkg/apis/rbac/v1beta1"
)

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
			Name:     "kubernikus:kubelet-bootstrap",
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
