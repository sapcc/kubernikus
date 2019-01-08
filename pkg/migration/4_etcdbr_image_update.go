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
	deploymentsClient := client.Extensions().Deployments(current.Namespace)
	deployment := fmt.Sprintf("%s-etcd", current.GetName())

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, err := deploymentsClient.Get(deployment, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("Deployment %s/%s not found: %v", current.Namespace, deployment, err)
		}

		i := 0
		for i < len(result.Spec.Template.Spec.Containers) {
			if strings.Contains(result.Spec.Template.Spec.Containers[i].Image, "sapcc/etcdbrctl") {
				result.Spec.Template.Spec.Containers[i].Image = "sapcc/etcdbrctl:0.4.1"
				_, err = deploymentsClient.Update(result)
				return err
			}
			i++
		}

		return nil
	})

	return err
}
