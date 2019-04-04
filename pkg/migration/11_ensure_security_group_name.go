package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

//This is primarily to fix very old (1.7) clusters
func EnsureSecurityGroupName(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {

	if current.Spec.Openstack.SecurityGroupName == "" {
		current.Spec.Openstack.SecurityGroupName = "default"
	}

	return nil
}
