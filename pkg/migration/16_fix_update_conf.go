package migration

import (
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

const (
	fixUpdateConf = `#!/bin/bash
cat <<EOF > /etc/coreos/update.conf
REBOOT_STRATEGY="off"
EOF

/usr/bin/pkill update_engine
/usr/bin/pkill locksmithd
`
)

func FixUpdateConf(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	client, err := clients.Satellites.ClientFor(current)
	if err != nil {
		return err
	}

	return ApplySuppository(fixUpdateConf, client)
}
