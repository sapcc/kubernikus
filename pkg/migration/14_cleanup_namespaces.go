package migration

import (
	"strings"

	"github.com/pkg/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

func CleanupSuppositoryNamespaces(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	client, err := clients.Satellites.ClientFor(current)
	if err != nil {
		return err
	}

	namespaces, err := client.CoreV1().Namespaces().List(meta.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to list namespaces")
	}
	for _, n := range namespaces.Items {
		if strings.HasPrefix(n.Name, "kubernikus-suppository-") {
			if err := client.CoreV1().Namespaces().Delete(n.Name, &meta.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "Failed to clean-up leftover suppository namespace")
			}
		}
	}

	return nil
}
