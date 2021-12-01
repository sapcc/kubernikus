package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/util"
)

// The swift audit backend needs to know the project domain id und user domain id
// due to a bug in the auth of its OpenStack provider (so project domain name) cannot
// be used.
func KlusterSecretOpenStackIds(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) error {
	secret, err := util.KlusterSecret(clients.Kubernetes, current)
	if err != nil {
		return err
	}
	if secret.AuditUserDomainID != "" && secret.AuditProjectDomainID != "" {
		return nil
	}
	adminClient, err := factories.Openstack.AdminClient()
	if err != nil {
		return err
	}
	domainNameByProject, err := adminClient.GetDomainNameByProject(secret.Openstack.ProjectID)
	if err != nil {
		return err
	}
	userDomainID, err := adminClient.GetDomainID("kubernikus")
	if err != nil {
		return err
	}
	projectDomainID, err := adminClient.GetDomainID(domainNameByProject)
	if err != nil {
		return err
	}
	secret.AuditUserDomainID = userDomainID
	secret.AuditProjectDomainID = projectDomainID
	err = util.UpdateKlusterSecret(clients.Kubernetes, current, secret)
	if err != nil {
		return err
	}
	return nil
}
