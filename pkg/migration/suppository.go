package migration

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
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

	namespace, err := client.CoreV1().Namespaces().Create(namespaceSpec)
	if err != nil {
		return errors.Wrap(err, "Failed to create namespace")
	}

	defer func() {
		client.CoreV1().Namespaces().Delete(namespace.Name, &meta.DeleteOptions{})
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

	if _, err := client.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil {
		return errors.Wrap(err, "Failed to create ClusterRoleBinding")
	}
	defer func() {
		client.RbacV1beta1().ClusterRoleBindings().Delete("kubernikus:suppository", &meta.DeleteOptions{})
	}()

	configMap := &core.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Name: "scripts",
		},
		Data: map[string]string{
			"migration.sh": script,
		},
	}

	if _, err := client.CoreV1().ConfigMaps(namespace.Name).Create(configMap); err != nil {
		return errors.Wrap(err, "Failed to create ConfigMap")
	}

	null := int64(0)
	yes := true
	daemonset := &extensions.DaemonSet{
		ObjectMeta: meta.ObjectMeta{
			Name: "kubernikus-suppository",
		},
		Spec: extensions.DaemonSetSpec{
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

	if _, err := client.ExtensionsV1beta1().DaemonSets(namespace.Name).Create(daemonset); err != nil {
		return errors.Wrap(err, "Failed to create Daemonset")
	}

	pods := informers.NewFilteredSharedInformerFactory(client, 1*time.Minute, namespace.Name, nil).Core().V1().Pods().Lister()

	wait.PollImmediate(1*time.Second, 2*time.Minute, func() (done bool, err error) {
		pods, err := pods.List(labels.Everything())
		if err != nil {
			return false, err
		}

		running := 0
		for _, pod := range pods {
			switch pod.Status.Phase {
			case core.PodRunning:
				running++
			case core.PodFailed:
				return false, fmt.Errorf("Failed to create a Pod: %v", pod.Status.Reason)
			}
		}

		return running == len(pods), nil
	})

	return nil
}
