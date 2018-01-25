package migration

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/sapcc/kubernikus/pkg/generated/clientset"
	listers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/migration"
	"github.com/sapcc/kubernikus/pkg/util"
)

func UpdateMigrationStatus(client clientset.Interface, lister listers_kubernikus.KlusterLister) error {
	klusters, err := lister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("Failed to list klusters: %s", err)
	}
	for _, kluster := range klusters {
		if migration.MigrationsPending(kluster) {
			//Update migration status
			err := util.UpdateKlusterMigrationStatus(client.Kubernikus(), kluster, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
