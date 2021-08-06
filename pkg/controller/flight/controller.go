package flight

import (
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/base"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

// =================================================================================
// FlightControl
// =================================================================================
//
// This controller takes care about Kluster health. It looks for obvious
// problems and tries to repair them.
//
// Currently implemented are the following helpers. See docs/controllers.md for
// more in depth explanation why these are required.
//
//
// Delete Incompletely Spawned Instances:
//
// It deletes Nodes that didn't manage to register within 10m after initial
// creation. This is a workaround for DHCP/DVS (latency) issues.  In effect it
// will delete the incompletely spawned node and launch control will ramp it
// back up.
//
//
// Delete Errored Instances:
//
// It deletes Nodes that are in state "error". This can have various causes. It
// frequently happens when instances are interrupted with another instance
// action while being spawned. Most interesstingly a Node also goes into
// "error" if the creation takes longer than the validity of the Keystone token
// used to create the VM. Instance create is a sequence of actions that happen
// sequentially in the same Nova request. If those take too long the next
// action in line will fail with an authentication error. This sets the VM into
// "error" state.
// FlightControl will pick these up and delete them.  GroundControl on the
// other hand ignores errored instances and just creates additional nodes. This
// leads to quota exhaustion and left-over instances. This could be the ultimate
// battle.
//
//
// Ensure Pod-to-Pod Communication via Security Group Rules:
//
// It ensures tcp/udp/icmp rules exist in the security group defined during
// kluster creation. The rules explicitly allow all pod-to-pod communication.
// This is a workaround for Neutron missing the side-channel security group
// events.
//
//
// Ensure Nodes belong to the security group:
//
// It ensures each Nodes is member of the security group defined in the kluster
// spec. This ensures missing security groups due to whatever reason are again
// added to the node.

type FlightController struct {
	Factory FlightReconcilerFactory
	Logger  log.Logger
}

func NewController(threadiness int, factories config.Factories, clients config.Clients, recorder record.EventRecorder, logger log.Logger) base.Controller {

	logger = log.With(logger, "controller", "flight")
	factory := NewFlightReconcilerFactory(factories.Openstack, clients.Kubernetes, factories.NodesObservatory.NodeInformer(), recorder, logger)

	var controller base.Reconciler
	controller = &FlightController{factory, logger}

	return base.NewController(threadiness, factories, controller, logger, nil, "flight")
}

func (d *FlightController) Reconcile(kluster *v1.Kluster) (bool, error) {
	//Skip klusters not in state running
	if kluster.Status.Phase != models.KlusterPhaseRunning {
		return false, nil
	}
	//Skip flight controller for klusters without cloudprovider
	if kluster.Spec.NoCloud {
		return false, nil
	}

	reconciler, err := d.Factory.FlightReconciler(kluster)
	if err != nil {
		return false, err
	}

	reconciler.EnsureKubernikusRulesInSecurityGroup()
	reconciler.EnsureInstanceSecurityGroupAssignment()
	reconciler.DeleteIncompletelySpawnedInstances()
	reconciler.DeleteErroredInstances()
	reconciler.EnsureServiceUserRoles()
	reconciler.EnsureNodeMetadataAndTags()

	return false, nil
}
