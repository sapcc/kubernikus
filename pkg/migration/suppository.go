package migration

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	image = "alpine:latest"
	pause = "gcr.io/google-containers/pause:latest"

	chrootScript = `set -x
cp /usr/local/scripts/migration.sh /host/tmp/
chmod +x /host/tmp/migration.sh
chroot /host /tmp/migration.sh
rm /host/tmp/migration.sh`
)

// ApplySuppository runs a script as a daemonset on each node. Then it self-destructs
func ApplySuppository(script string, client kubernetes.Interface) error {
	namespaceSpec := &core.Namespace{
		ObjectMeta: meta.ObjectMeta{
			GenerateName: "kubernikus-suppository-",
		},
	}

	// cleanup
	namespaces, err := client.CoreV1().Namespaces().List(context.TODO(), meta.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to list namespaces")
	}
	for _, n := range namespaces.Items {
		if strings.HasPrefix(n.Name, "kubernikus-suppository-") {
			if err := client.CoreV1().Namespaces().Delete(context.TODO(), n.Name, meta.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "Failed to clean-up leftover suppository namespace")
			}
		}
	}

	namespace, err := client.CoreV1().Namespaces().Create(context.TODO(), namespaceSpec, meta.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to create namespace")
	}
	defer func() {
		client.CoreV1().Namespaces().Delete(context.TODO(), namespace.Name, meta.DeleteOptions{})
	}()

	clusterRoleBinding := &rbac.ClusterRoleBinding{
		ObjectMeta: meta.ObjectMeta{
			Name: "kubernikus:suppository",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      "default",
				Namespace: namespace.Name,
			},
		},
	}

	if _, err := client.RbacV1beta1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, meta.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create RBAC clusterrolebinding")
		}

		if _, err := client.RbacV1beta1().ClusterRoleBindings().Update(context.TODO(), clusterRoleBinding, meta.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "unable to update RBAC clusterrolebinding")
		}
	}
	defer func() {
		client.RbacV1beta1().ClusterRoleBindings().Delete(context.TODO(), "kubernikus:suppository", meta.DeleteOptions{})
	}()

	configMap := &core.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Name: "scripts",
		},
		Data: map[string]string{
			"migration.sh": script,
		},
	}

	if _, err := client.CoreV1().ConfigMaps(namespace.Name).Create(context.TODO(), configMap, meta.CreateOptions{}); err != nil {
		return errors.Wrap(err, "Failed to create ConfigMap")
	}

	null := int64(0)
	yes := true
	daemonset := &apps.DaemonSet{
		ObjectMeta: meta.ObjectMeta{
			Name: "kubernikus-suppository",
		},
		Spec: apps.DaemonSetSpec{
			Selector: &meta.LabelSelector{
				MatchLabels: map[string]string{"app": "kubernikus-suppository"},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Name:      "kubernikus-suppository",
					Namespace: namespace.Name,
					Labels:    map[string]string{"app": "kubernikus-suppository"},
				},
				Spec: core.PodSpec{
					TerminationGracePeriodSeconds: &null,
					HostPID:                       true,
					InitContainers: []core.Container{
						{
							Name:  "init",
							Image: image,
							SecurityContext: &core.SecurityContext{
								Privileged: &yes,
							},
							Command: []string{"/bin/sh", "-c", chrootScript},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      "host",
									MountPath: "/host",
								},
								{
									Name:      "scripts",
									MountPath: "/usr/local/scripts",
								},
							},
						},
					},
					Containers: []core.Container{
						{
							Name:  "pause",
							Image: pause,
						},
					},
					Volumes: []core.Volume{
						{
							Name: "host",
							VolumeSource: core.VolumeSource{
								HostPath: &core.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
						{
							Name: "scripts",
							VolumeSource: core.VolumeSource{
								ConfigMap: &core.ConfigMapVolumeSource{
									LocalObjectReference: core.LocalObjectReference{
										Name: "scripts",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	created, err := client.AppsV1().DaemonSets(namespace.Name).Create(context.TODO(), daemonset, meta.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to create Daemonset")
	}

	wait.PollImmediate(5*time.Second, 5*time.Minute, func() (done bool, err error) { //nolint:staticcheck
		observed, err := client.AppsV1().DaemonSets(namespace.Name).Get(context.TODO(), "kubernikus-suppository", meta.GetOptions{})
		if err != nil {
			return false, err
		}

		if created.Generation != observed.Status.ObservedGeneration {
			return false, nil
		}

		return observed.Status.DesiredNumberScheduled == observed.Status.NumberReady, nil
	})

	return nil
}
