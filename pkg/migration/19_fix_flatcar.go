package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

const (
	fixFlannelOnFlatcar = `#!/bin/bash
mkdir -p /var/lib/coreos
`
)

func FixFlannelOnFlatcar(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	client, err := clients.Satellites.ClientFor(current)
	if err != nil {
		return err
	}

	return ApplySuppository(fixFlannelOnFlatcar, client)
}
