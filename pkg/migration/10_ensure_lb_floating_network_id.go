package migration

import (
	"fmt"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

// This migration is intended to run before we enable upgrades.
// It serves to purposes:
// 1. Add Spec.Version for 1.7 Klusters.
// 2. Update Spec.Version to match status for klusters we upgraded manually
func EnsureLBFloatingNetworkID(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {

	if current.Spec.Openstack.LBFloatingNetworkID == "" {
		client, err := factories.Openstack.ProjectAdminClientFor(current.Account())
		if err != nil {
			return err
		}
		md, err := client.GetMetadata()
		if err != nil {
			return err
		}
		for _, router := range md.Routers {
			if router.ID == current.Spec.Openstack.RouterID {
				if router.ExternalNetworkID == "" {
					return fmt.Errorf("Router %s has no external network ID", router.ID)
				}
				current.Spec.Openstack.LBFloatingNetworkID = router.ExternalNetworkID
				return nil
			}
		}
		return fmt.Errorf("Couldn't find router %s", current.Spec.Openstack.RouterID)
	}

	return nil
}
