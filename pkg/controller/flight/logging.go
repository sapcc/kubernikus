package flight

import (
	"fmt"
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

func (f *LoggingFlightReconciler) DeleteErroredInstances() []string {
	ids := f.Reconciler.DeleteErroredInstances()
	if len(ids) > 0 {
		f.Logger.Log(
			"msg", "deleted errored instances",
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

func (f *LoggingFlightReconciler) EnsureServiceUserRoles() []string {
	createdRoles := f.Reconciler.EnsureServiceUserRoles()
	if len(createdRoles) > 0 {
		f.Logger.Log(
			"msg", "created missing service user roles",
			"roles", fmt.Sprintf("%v", createdRoles),
			"v", 2,
		)
	}
	return createdRoles
}

func (f *LoggingFlightReconciler) EnsureNodeTags() []string {
	addedTags := f.Reconciler.EnsureNodeTags()
	if len(addedTags) > 0 {
		f.Logger.Log(
			"msg", "added missing node tags",
			"nodes", fmt.Sprintf("%v", addedTags),
			"v", 2,
		)
	}
	return addedTags
}

func (f *LoggingFlightReconciler) EnsureNodeMetadata() []string {
	addedMetadata := f.Reconciler.EnsureNodeMetadata()
	if len(addedMetadata) > 0 {
		f.Logger.Log(
			"msg", "added missing metadata to nodes",
			"nodes", fmt.Sprintf("%v", addedMetadata),
			"v", 2,
		)
	}
	return addedMetadata
}
