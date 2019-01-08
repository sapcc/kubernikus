package migration

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
)

func UpdateEtcdBackupContainerImage(rawKluster []byte, current *v1.Kluster, client kubernetes.Interface, openstackFactory openstack.SharedOpenstackClientFactory) (err error) {
	configClient := client.CoreV1().ConfigMaps(current.Namespace)
	config := fmt.Sprintf("%s-etcd", current.GetName())

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := configClient.Get(config, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("Error getting configmap %s/%s: %v", current.Namespace, config, err)
		}

		// Old klusters don't have an etcd configmap, ignore them
		if bs, ok := result.Data["bootstrap.sh"]; ok {
			if !strings.Contains(bs, "new.etcd") {
				bs = strings.Replace(bs, "#!/bin/sh", "#!/bin/sh\nif [ ! -d /var/lib/etcd/new.etcd ]; then\n    mkdir /var/lib/etcd/new.etcd\nfi\nif [ -d /var/lib/etcd/member ]; then\n    mv /var/lib/etcd/member /var/lib/etcd/new.etcd/member\nfi", 1)
				bs = strings.Replace(bs, "--data-dir=/var/lib/etcd", "--data-dir=/var/lib/etcd/new.etcd", 1)
				result.Data["bootstrap.sh"] = bs
				_, err := configClient.Update(result)
				return err
			}
		}

		return nil
	})

	deploymentsClient := client.Extensions().Deployments(current.Namespace)
	deployment := fmt.Sprintf("%s-etcd", current.GetName())
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := deploymentsClient.Get(deployment, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("Error getting etcd deployment %s/%s: %v", current.Namespace, deployment, err)
		}

		i := 0
		for i < len(result.Spec.Template.Spec.Containers) {
			if strings.Contains(result.Spec.Template.Spec.Containers[i].Image, "sapcc/etcdbrctl") {
				result.Spec.Template.Spec.Containers[i].Image = "sapcc/etcdbrctl:0.4.1"
				result.Spec.Template.Spec.Containers[i].Command = []string{
					"etcdbrctl",
					"server",
					"--schedule=15 * * * *",
					"--max-backups=168",
					"--data-dir=/var/lib/etcd/new.etcd",
					"--insecure-transport=true",
					"--storage-provider=Swift",
					"--delta-snapshot-period-seconds=10",
					"--garbage-collection-period-seconds=300",
					"--garbage-collection-policy=LimitBased",
				}
				_, err = deploymentsClient.Update(result)
				return err
			}
			i++
		}

		return nil
	})

	return err
}
