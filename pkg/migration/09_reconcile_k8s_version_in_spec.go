package migration

import (
	"errors"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

// This migration is intended to run before we enable upgrades.
// It serves to purposes:
// 1. Add Spec.Version for 1.7 Klusters.
// 2. Update Spec.Version to match status for klusters we upgraded manually
func ReconcileK8SVersionInSpec(rawKluster []byte, current *v1.Kluster, clients config.Clients, factories config.Factories) (err error) {

	if current.Spec.Version != current.Status.ApiserverVersion {
		if current.Status.ApiserverVersion == "" {
			return errors.New("No kubernetes version found in status")
		}
		current.Spec.Version = current.Status.ApiserverVersion
	}

	return
}
