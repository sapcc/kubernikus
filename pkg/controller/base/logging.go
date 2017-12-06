package base

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

var RECONCILLIATION_COUNTER = 0

type LoggingReconciler struct {
	Reconciler Reconciler
	Logger     log.Logger
}

func (r *LoggingReconciler) Reconcile(kluster *v1.Kluster) (requeue bool, err error) {
	defer func(begin time.Time) {
		r.Logger.Log(
			"msg", "reconciled kluster",
			"kluster", kluster.Spec.Name,
			"project", kluster.Account(),
			"requeue", requeue,
			"took", time.Since(begin),
			"v", 1,
			"err", err)
	}(time.Now())
	return r.Reconciler.Reconcile(kluster)
}
