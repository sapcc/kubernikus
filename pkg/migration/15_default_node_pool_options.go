package migration

import (
	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

func ReconcileNodePoolConfigDefaults(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	for i := range current.Spec.NodePools {
		allowReboot := true
		allowReplace := true
		if current.Spec.NodePools[i].Config == nil {
			current.Spec.NodePools[i].Config = &models.NodePoolConfig{
				AllowReboot:  &allowReboot,
				AllowReplace: &allowReplace,
			}
			continue
		}

		if current.Spec.NodePools[i].Config.AllowReboot == nil {
			current.Spec.NodePools[i].Config.AllowReboot = &allowReboot
		}

		if current.Spec.NodePools[i].Config.AllowReplace == nil {
			current.Spec.NodePools[i].Config.AllowReplace = &allowReplace
		}
	}
	return
}
