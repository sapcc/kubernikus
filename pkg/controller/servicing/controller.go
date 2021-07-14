package servicing

import (
	"time"

	"github.com/pkg/errors"

	"github.com/go-kit/kit/log"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

const (
	// AnnotationServicingSafeguard must be set to enable servicing
	AnnotationServicingSafeguard = "kubernikus.cloud.sap/servicing"
)

var (
	// Now is a poor-man's facility to change time during testing
	Now = time.Now
)

// Controller periodically checks for nodes requiting updates or upgrades
//
// This controller handles node upgrades when the Kubernetes or CoreOS versions
// are changed. It gracefully drains nodes before performing any action.
//
// For Kubernetes upgrades the strategy is to replace the fleet by terminating
// the nodes. CoreOS updates are handled by a soft reboot.
//
// In order to allow the payload to settle only a single node per cluster is
// processed at a time. Between updates there's a 1h grace period.
//
// In case any node in the cluster is unhealthy the upgrades are skipped. This
// is to safeguard against failed upgrades destroying the universe.
//
// For rollout and testing purposed the node upgrades are disabled by default.
// They can manually be enabled by setting the node annotaion:
//
//    kubernikus.cloud.sap/servicing=true
//
type Controller struct {
	Logger     log.Logger
	Reconciler ReconcilerFactory
}

// NewController is a helper to create a Servicing Controller instance
func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {
	logger = log.With(logger, "controller", "servicing")

	var controller base.Reconciler
	controller = &Controller{
		Logger:     logger,
		Reconciler: NewKlusterReconcilerFactory(logger, recorder, factories, clients),
	}

	RegisterServicingNodesCollector(logger, factories)

	return base.NewController(threadiness, factories, controller, logger, nil, "servicing")
}

// Reconcile checks a kluster for node updates
func (d *Controller) Reconcile(k *v1.Kluster) (requeue bool, err error) {
	// Disabled until node templates are fixed
	return false, nil

	//Skip klusters not in state running
	if k.Status.Phase != models.KlusterPhaseRunning {
		return false, nil
	}
	reconciler, err := d.Reconciler.Make(k)
	if err != nil {
		d.Logger.Log("msg", "skippig upgrades. Internal server error.", "err", err)
		return true, errors.Wrap(err, "Couldn't make Servicing Reconciler.")
	}

	return false, reconciler.Do()
}
