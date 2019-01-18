package migration

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	"github.com/sapcc/kubernikus/pkg/util"
)

func MigrateKlusterSecret(rawKluster []byte, current *v1.Kluster, client kubernetes.Interface, openstackFactory openstack.SharedOpenstackClientFactory) (err error) {

	oldSecret, err := client.CoreV1().Secrets(current.Namespace).Get(current.Name, meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	if _, err := util.EnsureKlusterSecret(client, current); err != nil {
		return err
	}

	newSecret, err := client.CoreV1().Secrets(current.Namespace).Get(current.Name+"-secret", meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	newSecret.Data = oldSecret.Data

	_, err = client.CoreV1().Secrets(current.Namespace).Update(newSecret)
	return err
}
