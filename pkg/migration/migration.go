package migration

import (
	"fmt"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
)

var defaultRegistry Registry

// Migration describes an individual migration step.
// The klusterRaw and kluster contain the to be migrate kluster.
// The function is expected to modify the kluster accordingly, changed object is persisted
// automatically after the handler returns with no error.
// The kubernetes client can be used to modify other things (e.g. kluster secret)
type Migration func(klusterRaw []byte, kluster *v1.Kluster, clients config.Clients, factories config.Factories) (err error)

//Latest returns to latest spec version available
func Latest() int {
	return defaultRegistry.Latest()
}

//MigrationsPending returns true if a kluster needs to be migrated
func MigrationsPending(kluster *v1.Kluster) bool {
	return defaultRegistry.MigrationsPending(kluster)
}

//Migrate a kluster to the most recent spec version
func Migrate(k *v1.Kluster, clients config.Clients, factories config.Factories) error {
	return defaultRegistry.Migrate(k, clients, factories)
}

//Registry manages an ordered list of migration steps
type Registry struct {
	migrations []Migration
}

//AddMigration appends a migration to the list
func (r *Registry) AddMigration(m Migration) {
	r.migrations = append(r.migrations, m)
}

func (r Registry) Latest() int {
	return len(r.migrations)
}

func (r Registry) MigrationsPending(kluster *v1.Kluster) bool {
	return int(kluster.Status.SpecVersion) < r.Latest()
}

func (r *Registry) Migrate(k *v1.Kluster, clients config.Clients, factories config.Factories) error {
	klusterVersion := int(k.Status.SpecVersion)
	if klusterVersion >= r.Latest() {
		return nil
	}

	kluster := k.DeepCopy()
	var err error
	for idx := klusterVersion; idx < r.Latest(); idx++ {
		migration := r.migrations[idx]
		version := idx + 1
		if kluster, err = migrateKluster(kluster, version, migration, clients, factories); err != nil {
			return fmt.Errorf("Error running migration %d: %s", version, err)
		}
	}
	return nil
}

func migrateKluster(kluster *v1.Kluster, version int, migration Migration, clients config.Clients, factories config.Factories) (*v1.Kluster, error) {
	var rawData []byte
	var err error

	//TODO: Don't import fake pkg outside of test code
	if _, ok := clients.Kubernikus.(*kubernikusfake.Clientset); !ok {
		request := clients.Kubernikus.Kubernikus().RESTClient().Get().Namespace(kluster.Namespace).Resource("klusters").Name(kluster.Name)
		if rawData, err = request.DoRaw(); err != nil {
			return nil, err
		}
	}

	if err = migration(rawData, kluster, clients, factories); err != nil {
		return nil, err
	}
	kluster.Status.SpecVersion = int64(version)
	return clients.Kubernikus.Kubernikus().Klusters(kluster.Namespace).Update(kluster)
}
