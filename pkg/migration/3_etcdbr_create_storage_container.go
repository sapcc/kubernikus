package migration

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
)

func CreateEtcdBackupStorageContainer(rawKluster []byte, current *v1.Kluster, client kubernetes.Interface, openstackFactory openstack.SharedOpenstackClientFactory) (err error) {
	secret, err := client.CoreV1().Secrets(current.GetNamespace()).Get(current.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	username, ok := secret.Data["openstack-username"]
	if !ok {
		return errors.New("openstack username secret not set")
	}

	domain, ok := secret.Data["openstack-domain-name"]
	if !ok {
		return errors.New("openstack domain name secret not set")
	}

	adminClient, err := openstackFactory.AdminClient()
	if err != nil {
		return err
	}

	err = adminClient.CreateStorageContainer(
		current.Spec.Openstack.ProjectID,
		etcd_util.DefaultStorageContainer(current),
		string(username),
		string(domain),
	)

	return err
}
