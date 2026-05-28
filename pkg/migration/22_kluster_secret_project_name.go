package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/util"
)

// Backfill ProjectName on the kluster Secret for clusters created before
// ground.go started populating it. Required by etcdbrctl's swift creds file.
func KlusterSecretProjectName(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) error {
	if current.Spec.NoCloud {
		return nil
	}
	secret, err := util.KlusterSecret(clients.Kubernetes, current)
	if err != nil {
		return err
	}
	if secret.ProjectName != "" {
		return nil
	}
	adminClient, err := factories.Openstack.AdminClient()
	if err != nil {
		return err
	}
	projectName, err := adminClient.GetProjectName(secret.ProjectID)
	if err != nil {
		return err
	}
	secret.ProjectName = projectName
	return util.UpdateKlusterSecret(clients.Kubernetes, current, secret)
}
