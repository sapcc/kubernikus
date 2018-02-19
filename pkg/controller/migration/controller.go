package migration

import (
	"github.com/go-kit/kit/log"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/migration"
	"github.com/sapcc/kubernikus/pkg/util"
)

const (
	MigrationFailed = "MigrationFailed"
)

type MigrationReconciler struct {
	config.Clients

	Recorder record.EventRecorder
	Logger   log.Logger
}

func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger,
		"controller", "migration")

	var reconciler base.Reconciler
	reconciler = &MigrationReconciler{clients, recorder, logger}

	return base.NewController(threadiness, factories, reconciler, logger, nil)
}

func (mr *MigrationReconciler) Reconcile(kluster *v1.Kluster) (bool, error) {

	//We only care about klustes with pending migrations
	if !migration.MigrationsPending(kluster) {
		// Ensure the kluster migration status is up to date
		return false, util.UpdateKlusterMigrationStatus(mr.Kubernikus.Kubernikus(), kluster, false)
	}

	//Ensure pending migrations are reflected in the status
	if err := util.UpdateKlusterMigrationStatus(mr.Kubernikus.Kubernikus(), kluster, true); err != nil {
		return false, err
	}

	err := migration.Migrate(kluster, mr.Kubernetes, mr.Kubernikus)
	mr.Logger.Log(
		"msg", "Migrating spec",
		"kluster", kluster.Name,
		"from", int(kluster.Status.SpecVersion),
		"to", migration.Latest(),
		"err", err,
	)
	if err != nil {
		mr.Recorder.Event(kluster, api_v1.EventTypeWarning, MigrationFailed, err.Error())
		return false, err
	}
	//Clear the klusters migration status as migrations are applied successfully
	util.UpdateKlusterMigrationStatus(mr.Kubernikus.Kubernikus(), kluster, false)

	return false, nil
}
