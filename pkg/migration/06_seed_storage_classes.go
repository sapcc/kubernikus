package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
)

func SeedCinderStorageClasses(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	kubernetes, err := clients.Satellites.ClientFor(current)
	if err != nil {
		return err
	}

	openstack, err := factories.Openstack.ProjectAdminClientFor(current.Account())
	if err != nil {
		return err
	}

	return ground.SeedCinderStorageClasses(kubernetes, openstack, false)
}
