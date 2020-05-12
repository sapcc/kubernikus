package bootstrap

import (
	"bytes"
	"html/template"

	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
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
	if _, err := client.CoreV1().ServiceAccounts(sa.ObjectMeta.Namespace).Create(sa); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create serviceaccount")
		}
	}
	return nil
}

func CreateOrUpdateDeployment(client clientset.Interface, deploy *extensions.Deployment) error {
	if _, err := client.ExtensionsV1beta1().Deployments(deploy.ObjectMeta.Namespace).Create(deploy); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create deployment")
		}

		if _, err := client.ExtensionsV1beta1().Deployments(deploy.ObjectMeta.Namespace).Update(deploy); err != nil {
			return errors.Wrap(err, "unable to update deployment")
		}
	}
	return nil
}

func CreateOrUpdateDeployment116(client clientset.Interface, deploy *apps.Deployment) error {
	if _, err := client.AppsV1().Deployments(deploy.ObjectMeta.Namespace).Create(deploy); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create deployment")
		}

		if _, err := client.AppsV1().Deployments(deploy.ObjectMeta.Namespace).Update(deploy); err != nil {
			return errors.Wrap(err, "unable to update deployment")
		}
	}
	return nil
}

func CreateOrUpdateService(client clientset.Interface, service *v1.Service) error {

	if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Get(service.Name, metav1.GetOptions{}); err == nil {
		if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Update(service); err != nil {
			return errors.Wrapf(err, "unable to create/update the kube-dns service")
		}
	} else {
		if _, err := client.CoreV1().Services(metav1.NamespaceSystem).Create(service); err != nil {
			return errors.Wrap(err, "unable to create a new kube-dns service")
		}
	}
	return nil
}

func CreateOrUpdateConfigMap(client clientset.Interface, configmap *v1.ConfigMap) error {
	if _, err := client.CoreV1().ConfigMaps(configmap.ObjectMeta.Namespace).Create(configmap); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create configmap")
		}

		if _, err := client.CoreV1().ConfigMaps(configmap.ObjectMeta.Namespace).Update(configmap); err != nil {
			return errors.Wrap(err, "unable to update configmap")
		}
	}
	return nil
}

func CreateOrUpdateDaemonset(client clientset.Interface, daemonset *apps.DaemonSet) error {
	if _, err := client.Apps().DaemonSets(daemonset.ObjectMeta.Namespace).Create(daemonset); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create daemonset")
		}

		if _, err := client.Apps().DaemonSets(daemonset.ObjectMeta.Namespace).Update(daemonset); err != nil {
			return errors.Wrap(err, "unable to update daemonset")
		}
	}
	return nil
}

func CreateOrUpdateClusterRoleBinding(client clientset.Interface, clusterRoleBinding *rbac.ClusterRoleBinding) error {
	if _, err := client.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC clusterrolebinding")
		}

		if _, err := client.RbacV1beta1().ClusterRoleBindings().Update(clusterRoleBinding); err != nil {
			return errors.Wrap(err, "unable to update RBAC clusterrolebinding")
		}
	}
	return nil
}

func CreateOrUpdateRoleBinding(client clientset.Interface, roleBinding *rbac.RoleBinding) error {
	if _, err := client.RbacV1beta1().RoleBindings(roleBinding.Namespace).Create(roleBinding); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC rolebinding")
		}

		if _, err := client.RbacV1beta1().RoleBindings(roleBinding.Namespace).Update(roleBinding); err != nil {
			return errors.Wrap(err, "unable to update RBAC rolebinding")
		}
	}
	return nil
}

func CreateOrUpdateClusterRole(client clientset.Interface, clusterRole *rbac.ClusterRole) error {
	if _, err := client.RbacV1beta1().ClusterRoles().Create(clusterRole); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC clusterrole")
		}

		if _, err := client.RbacV1beta1().ClusterRoles().Update(clusterRole); err != nil {
			return errors.Wrap(err, "unable to update RBAC clusterrole")
		}
	}
	return nil
}
