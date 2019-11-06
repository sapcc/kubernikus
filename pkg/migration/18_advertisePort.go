package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

func ReconcileAdvertisePortConfigDefault(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	if current.Spec.AdvertisePort == 0 {
		current.Spec.AdvertisePort = 6443
	}
	return
}
