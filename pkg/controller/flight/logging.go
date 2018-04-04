package flight

import (
	"strings"

	"github.com/go-kit/kit/log"
)

type LoggingFlightReconciler struct {
	Reconciler FlightReconciler
	Logger     log.Logger
}

func (f *LoggingFlightReconciler) EnsureInstanceSecurityGroupAssignment() []string {
	ids := f.Reconciler.EnsureInstanceSecurityGroupAssignment()
	if len(ids) > 0 {
		f.Logger.Log(
			"msg", "added missing security group",
			"nodes", strings.Join(ids, ","),
			"v", 2,
		)
	}
	return ids
}

func (f *LoggingFlightReconciler) DeleteIncompletelySpawnedInstances() []string {
	ids := f.Reconciler.DeleteIncompletelySpawnedInstances()
	if len(ids) > 0 {
		f.Logger.Log(
			"msg", "deleted incompletely spawned instances",
			"nodes", strings.Join(ids, ","),
			"v", 2,
		)
	}
	return ids
}

func (f *LoggingFlightReconciler) EnsureKubernikusRuleInSecurityGroup() bool {
	ensured := f.Reconciler.EnsureKubernikusRuleInSecurityGroup()
	if ensured {
		f.Logger.Log(
			"msg", "added missing kubernikus security group",
			"v", 2,
		)
	}
	return ensured
}
