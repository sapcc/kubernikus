package migration

import (
	"fmt"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/ground"
)

func SeedKubernikusServiceAccount(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {
	kubernetes, err := clients.Satellites.ClientFor(current)
	if err != nil {
		return err
	}
	if err := ground.SeedKubernikusServiceAccount(kubernetes); err != nil {
		return fmt.Errorf("Failed to seed kubernikus service account: %w", err)
	}
	if err := ground.UpdateServiceAccountTokenInSecret(current, clients.Kubernetes, kubernetes); err != nil {
		return fmt.Errorf("Failed to update sa token in cluster secret: %w", err)
	}
	return nil
}
