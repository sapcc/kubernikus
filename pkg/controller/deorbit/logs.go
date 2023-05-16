package deorbit

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type LoggingDeorbiter struct {
	Deorbiter Deorbiter
	Logger    log.Logger
}

func (d *LoggingDeorbiter) DeleteServices() (deleted []core_v1.Service, err error) {
	defer func(begin time.Time) {
		list := make([]string, len(deleted))
		for i, v := range deleted {
			list[i] = fmt.Sprintf("%v/%v", v.Namespace, v.Name)
		}

		d.Logger.Log(
			"msg", "deleted services",
			"services", strings.Join(list, ", "),
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return d.Deorbiter.DeleteServices()
}

func (d *LoggingDeorbiter) DeleteSnapshots() (deleted []*unstructured.Unstructured, err error) {
	defer func(begin time.Time) {
		list := make([]string, len(deleted))
		for i, v := range deleted {
			list[i] = fmt.Sprintf("%v/%v", v.GetNamespace(), v.GetName())
		}

		d.Logger.Log(
			"msg", "deleted snapshots",
			"services", strings.Join(list, ", "),
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return d.Deorbiter.DeleteSnapshots()
}

func (d *LoggingDeorbiter) DeletePersistentVolumeClaims() (deleted []core_v1.PersistentVolumeClaim, err error) {
	defer func(begin time.Time) {
		list := make([]string, len(deleted))
		for i, v := range deleted {
			list[i] = fmt.Sprintf("%v/%v", v.Namespace, v.Name)
		}

		d.Logger.Log(
			"msg", "deleted pvcs",
			"pvcs", strings.Join(list, ", "),
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return d.Deorbiter.DeletePersistentVolumeClaims()
}

func (d *LoggingDeorbiter) WaitForSnapshotCleanUp() (err error) {
	defer func(begin time.Time) {
		d.Logger.Log(
			"msg", "waited for snapshot cleanup",
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return d.Deorbiter.WaitForSnapshotCleanUp()
}

func (d *LoggingDeorbiter) WaitForPersistentVolumeCleanup() (err error) {
	defer func(begin time.Time) {
		d.Logger.Log(
			"msg", "waited for pv cleanup",
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return d.Deorbiter.WaitForPersistentVolumeCleanup()
}

func (d *LoggingDeorbiter) WaitForServiceCleanup() (err error) {
	defer func(begin time.Time) {
		d.Logger.Log(
			"msg", "waited for service cleanup",
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return d.Deorbiter.WaitForServiceCleanup()
}

func (d *LoggingDeorbiter) SelfDestruct(reason SelfDestructReason) (err error) {
	defer func(begin time.Time) {
		d.Logger.Log(
			"msg", "initiated self-destruction",
			"reason", reason,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return d.Deorbiter.SelfDestruct(reason)
}

func (d *LoggingDeorbiter) IsAPIUnavailableTimeout() bool {
	return d.Deorbiter.IsAPIUnavailableTimeout()
}

func (d *LoggingDeorbiter) IsDeorbitHangingTimeout() bool {
	return d.Deorbiter.IsDeorbitHangingTimeout()
}
