package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

func Helm2to3(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) error {
	//this is a noop since we migrated the last existing deployments and ripped out support for helm2
	return nil
}
