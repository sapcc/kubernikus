package migration

import (
	"k8s.io/client-go/kubernetes"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/util"
	etcd_util "github.com/sapcc/kubernikus/pkg/util/etcd"
)

func CreateEtcdBackupStorageContainer(rawKluster []byte, current *v1.Kluster, client kubernetes.Interface, openstackFactory openstack.SharedOpenstackClientFactory) (err error) {
	secret, err := util.KlusterSecret(client, current)
	if err != nil {
		return err
	}

	adminClient, err := openstackFactory.AdminClient()
	if err != nil {
		return err
	}

	err = adminClient.CreateStorageContainer(
		current.Spec.Openstack.ProjectID,
		etcd_util.DefaultStorageContainer(current),
		secret.Openstack.Username,
		secret.Openstack.DomainName,
	)

	return err
}
