package migration

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/admin"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
)

func CreateEtcdBackupStorageContainer(rawKluster []byte, current *v1.Kluster, client kubernetes.Interface, adminClient admin.AdminClient) (err error) {
	secret, err := client.CoreV1().Secrets(current.GetNamespace()).Get(current.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	username := string(secret.Data["openstack-username"])
	domain := string(secret.Data["openstack-domain-name"])

	err = adminClient.CreateStorageContainer(
		current.Spec.Openstack.ProjectID,
		etcd_util.DefaultStorageContainer(current),
		username,
		domain,
	)

	return err
}
