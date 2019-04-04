package migration

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/util"
)

func MigrateKlusterSecret(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {

	oldSecret, err := clients.Kubernetes.CoreV1().Secrets(current.Namespace).Get(current.Name, meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	if _, err := util.EnsureKlusterSecret(clients.Kubernetes, current); err != nil {
		return err
	}

	newSecret, err := clients.Kubernetes.CoreV1().Secrets(current.Namespace).Get(current.Name+"-secret", meta_v1.GetOptions{})
	if err != nil {
		return err
	}
	newSecret.Data = oldSecret.Data

	_, err = clients.Kubernetes.CoreV1().Secrets(current.Namespace).Update(newSecret)
	return err
}
