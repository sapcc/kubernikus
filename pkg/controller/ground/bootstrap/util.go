package bootstrap

import (
	"bytes"
	"context"
	"html/template"

	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	rbac "k8s.io/api/rbac/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

func RenderManifest(strtmpl string, obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(strtmpl)
	if err != nil {
		return nil, errors.Wrap(err, "error when parsing template:")
	}
	err = tmpl.Execute(&buf, obj)
	if err != nil {
		return nil, errors.Wrap(err, "error when executing template:")
	}
	return buf.Bytes(), nil
}

func CreateOrUpdateServiceAccount(client clientset.Interface, sa *v1.ServiceAccount) error {
	if _, err := client.CoreV1().ServiceAccounts(sa.Namespace).Create(context.TODO(), sa, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create serviceaccount")
		}
	}
	return nil
}

func CreateOrUpdateDeployment(client clientset.Interface, deploy *extensions.Deployment) error {
	if _, err := client.ExtensionsV1beta1().Deployments(deploy.Namespace).Create(context.TODO(), deploy, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create deployment")
		}

		if _, err := client.ExtensionsV1beta1().Deployments(deploy.Namespace).Update(context.TODO(), deploy, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update deployment")
		}
	}
	return nil
}

func CreateOrUpdateDeployment116(client clientset.Interface, deploy *apps.Deployment) error {
	if _, err := client.AppsV1().Deployments(deploy.Namespace).Create(context.TODO(), deploy, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create deployment")
		}

		if _, err := client.AppsV1().Deployments(deploy.Namespace).Update(context.TODO(), deploy, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update deployment")
		}
	}
	return nil
}

func CreateOrUpdateService(client clientset.Interface, service *v1.Service) error {
	if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Get(context.TODO(), service.Name, metav1.GetOptions{}); err == nil {
		if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Update(context.TODO(), service, metav1.UpdateOptions{}); err != nil {
			return errors.Wrapf(err, "unable to create/update the kube-dns service")
		}
	} else {
		if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Create(context.TODO(), service, metav1.CreateOptions{}); err != nil {
			return errors.Wrap(err, "unable to create a new kube-dns service")
		}
	}
	return nil
}

func CreateOrUpdateConfigMap(client clientset.Interface, configmap *v1.ConfigMap) error {
	if _, err := client.CoreV1().ConfigMaps(configmap.Namespace).Create(context.TODO(), configmap, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create configmap")
		}

		if _, err := client.CoreV1().ConfigMaps(configmap.Namespace).Update(context.TODO(), configmap, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update configmap")
		}
	}
	return nil
}

func CreateOrUpdateDaemonset(client clientset.Interface, daemonset *apps.DaemonSet) error {
	if _, err := client.AppsV1().DaemonSets(daemonset.Namespace).Create(context.TODO(), daemonset, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create daemonset")
		}

		if _, err := client.AppsV1().DaemonSets(daemonset.Namespace).Update(context.TODO(), daemonset, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update daemonset")
		}
	}
	return nil
}

func CreateOrUpdateClusterRoleBinding(client clientset.Interface, clusterRoleBinding *rbac.ClusterRoleBinding) error {
	if _, err := client.RbacV1beta1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC clusterrolebinding")
		}

		if _, err := client.RbacV1beta1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC clusterrolebinding")
		}
	}
	return nil
}

func CreateOrUpdateClusterRoleBindingV1(client clientset.Interface, clusterRoleBinding *rbac_v1.ClusterRoleBinding) error {
	if _, err := client.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC clusterrolebinding")
		}

		if _, err := client.RbacV1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC clusterrolebinding")
		}
	}
	return nil
}

func CreateOrUpdateRoleBinding(client clientset.Interface, roleBinding *rbac.RoleBinding) error {
	if _, err := client.RbacV1beta1().RoleBindings(roleBinding.Namespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC rolebinding")
		}

		if _, err := client.RbacV1beta1().RoleBindings(roleBinding.Namespace).Update(context.TODO(), roleBinding, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC rolebinding")
		}
	}
	return nil
}

func CreateOrUpdateRoleBindingV1(client clientset.Interface, roleBinding *rbac_v1.RoleBinding) error {
	if _, err := client.RbacV1().RoleBindings(roleBinding.Namespace).Create(context.TODO(), roleBinding, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC rolebinding")
		}

		if _, err := client.RbacV1().RoleBindings(roleBinding.Namespace).Update(context.TODO(), roleBinding, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC rolebinding")
		}
	}
	return nil
}

func CreateOrUpdateRole(client clientset.Interface, role *rbac_v1.Role) error {
	if _, err := client.RbacV1().Roles(role.Namespace).Create(context.TODO(), role, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC role")
		}

		if _, err := client.RbacV1().Roles(role.Namespace).Update(context.TODO(), role, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC role")
		}
	}
	return nil
}

func CreateOrUpdateClusterRole(client clientset.Interface, clusterRole *rbac.ClusterRole) error {
	if _, err := client.RbacV1beta1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC clusterrole")
		}

		if _, err := client.RbacV1beta1().ClusterRoles().Update(context.TODO(), clusterRole, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC clusterrole")
		}
	}
	return nil
}

func CreateOrUpdateClusterRoleV1(client clientset.Interface, clusterRole *rbac_v1.ClusterRole) error {
	if _, err := client.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC clusterrole")
		}

		if _, err := client.RbacV1().ClusterRoles().Update(context.TODO(), clusterRole, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC clusterrole")
		}
	}
	return nil
}

func CreateOrUpdateStatefulSet(client clientset.Interface, statefulSet *apps.StatefulSet) error {
	if _, err := client.AppsV1().StatefulSets(statefulSet.Namespace).Create(context.TODO(), statefulSet, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create StatefulSet")
		}

		if _, err := client.AppsV1().StatefulSets(statefulSet.Namespace).Update(context.TODO(), statefulSet, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update StatefulSet")
		}
	}
	return nil
}

func CreateOrUpdateSecret(client clientset.Interface, secret *v1.Secret) error {
	if _, err := client.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create Secret")
		}

		if _, err := client.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update Secret")
		}
	}
	return nil
}
