package migration

import (
	"fmt"

	"github.com/aokoli/goutils"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
	"github.com/sapcc/kubernikus/pkg/util"
)

func AddDexSecretAndRoleBindings(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {

	// Secret
	apiSecret, err := util.KlusterSecret(clients.Kubernetes, current)
	if err != nil {
		return fmt.Errorf("Failed to serialize secret data: %s", err)
	}

	var randomPasswordChars = []rune("abcdefghjkmnpqrstuvwxABCDEFGHJKLMNPQRSTUVWX23456789")

	if apiSecret.DexClientSecret == "" {
		apiSecret.DexClientSecret, err = goutils.Random(16, 0, 0, true, true, randomPasswordChars...)
		if err != nil {
			return fmt.Errorf("Failed to generate dex client secret: %s", err)
		}
	}

	if apiSecret.DexStaticPassword == "" {
		apiSecret.DexStaticPassword, err = goutils.Random(16, 0, 0, true, true, randomPasswordChars...)
		if err != nil {
			return fmt.Errorf("Failed to generate dex static password: %s", err)
		}
	}

	if apiSecret.Openstack.ProjectDomainName == "" {

		domainFromLabel := current.Domain()

		if domainFromLabel == "" {
			domainFromOpenstack, err := clients.OpenstackAdmin.GetDomainNameByProject(apiSecret.ProjectID)
			if err != nil {
				return err
			}
			apiSecret.Openstack.ProjectDomainName = domainFromOpenstack
		} else {
			apiSecret.Openstack.ProjectDomainName = domainFromLabel
		}

	}

	err = util.UpdateKlusterSecret(clients.Kubernetes, current, apiSecret)
	if err != nil {
		return err
	}

	// Seed dex cluster role bindings
	kubernetes, err := clients.Satellites.ClientFor(current)
	if err != nil {
		return err
	}
	err = ground.SeedOpenStackClusterRoleBindings(kubernetes)
	if err != nil {
		return err
	}

	return nil
}
