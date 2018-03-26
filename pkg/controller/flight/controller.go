package flight

import (
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

// =================================================================================
//   FlightControl
// =================================================================================
//
// This controller takes care about Kluster health. It looks for obvious
// problems and tries to repair them.
//
// Currently implemented are the following helpers. See docs/controllers.md for more
// in depth explanation why these are required.
//
//
// Delete Incompletely Spawned Instances:
//
// It deletes Nodes that didn't manage to register within 10m after
// inital creation. This is a workaround for DHCP/DVS (latency) issues.  In effect
// it will delete the incompletely spawned node and launch control will ramp it
// back up.
//
// Ensure Pod-to-Pod Communication via Security Group Rules:
//
// It ensures tcp/udp/icmp rules exist in the security group defined during
// kluster creation. The rules explicitly allow all pod-to-pod
// communication. This is a workaround for Neutron missing the
// side-channel security group events.
//
//
// Ensure Nodes belong to the security group:
//
// It ensures each Nodes is member of the security group defined in
// the kluster spec. This ensures missing security groups due to whatever
// reason are again added to the node.

type FlightController struct {
	Factory FlightReconcilerFactory
	Logger  log.Logger
}

func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {

	logger = log.With(logger, "controller", "flight")
	factory := NewFlightReconcilerFactory(factories.Openstack, factories.NodesObservatory.NodeInformer(), recorder, logger)

	var controller base.Reconciler
	controller = &FlightController{factory, logger}

	return base.NewController(threadiness, factories, controller, logger, nil)
}

func (d *FlightController) Reconcile(kluster *v1.Kluster) (bool, error) {
	reconciler, err := d.Factory.FlightReconciler(kluster)
	if err != nil {
		return false, err
	}

	reconciler.EnsureKubernikusRuleInSecurityGroup()
	reconciler.EnsureInstanceSecurityGroupAssignment()
	reconciler.DeleteIncompletelySpawnedInstances()

	return false, nil
}
