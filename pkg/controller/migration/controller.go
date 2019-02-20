package migration

import (
	"github.com/go-kit/kit/log"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/migration"
	"github.com/sapcc/kubernikus/pkg/util"
)

const (
	MigrationFailed = "MigrationFailed"
)

type MigrationReconciler struct {
	Clients   config.Clients
	Factories config.Factories
	Recorder  record.EventRecorder
	Logger    log.Logger
}

func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger,
		"controller", "migration")

	var reconciler base.Reconciler
	reconciler = &MigrationReconciler{clients, factories, recorder, logger}

	return base.NewController(threadiness, factories, reconciler, logger, nil, "migration")
}

func (mr *MigrationReconciler) Reconcile(kluster *v1.Kluster) (bool, error) {

	//We only care about klusters with pending migrations
	if !migration.MigrationsPending(kluster) {
		// Ensure the kluster migration status is up to date
		return false, util.UpdateKlusterMigrationStatus(mr.Clients.Kubernikus.Kubernikus(), kluster, false)
	}

	//Ensure pending migrations are reflected in the status
	if err := util.UpdateKlusterMigrationStatus(mr.Clients.Kubernikus.Kubernikus(), kluster, true); err != nil {
		return false, err
	}

	// don't continue if the updated staus is not reflected yet, wait for the next reconciliation update (triggerd by the migration status update)
	if !kluster.Status.MigrationsPending {
		return false, nil
	}

	err := migration.Migrate(kluster, mr.Clients, mr.Factories)
	mr.Logger.Log(
		"msg", "Migrating spec",
		"kluster", kluster.Name,
		"from", int(kluster.Status.SpecVersion),
		"to", migration.Latest(),
		"err", err,
	)
	if err != nil {
		mr.Recorder.Event(kluster, api_v1.EventTypeWarning, MigrationFailed, err.Error())
		metrics.MigrationErrorsTotal.WithLabelValues(kluster.Name).Inc()
		return false, err
	}
	//Clear the klusters migration status as migrations are applied successfully
	util.UpdateKlusterMigrationStatus(mr.Clients.Kubernikus.Kubernikus(), kluster, false)

	return false, nil
}
