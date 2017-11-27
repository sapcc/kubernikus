package base

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

var RECONCILLIATION_COUNTER = 0

type LoggingReconciler struct {
	Reconciler Reconciler
	Logger     log.Logger
}

type EventingReconciler struct {
	Reconciler
}

type InstrumentedReconciler struct {
	Reconciler
}

func (r *InstrumentedReconciler) Reconcile(kluster *v1.Kluster) (requeue bool, err error) {
	defer func() {
		RECONCILLIATION_COUNTER = RECONCILLIATION_COUNTER + 1
		fmt.Printf("Metrics: Reconciled %v kluster\n", RECONCILLIATION_COUNTER)
	}()
	return r.Reconciler.Reconcile(kluster)
}

func (r *EventingReconciler) Reconcile(kluster *v1.Kluster) (requeue bool, err error) {
	fmt.Printf("EVENT: Reconciled %v\n", kluster.Name)
	return r.Reconciler.Reconcile(kluster)
}

func (r *LoggingReconciler) Reconcile(kluster *v1.Kluster) (requeue bool, err error) {
	defer func(begin time.Time) {
		r.Logger.Log(
			"msg", "reconciled kluster",
			"kluster", kluster.Name,
			"requeue", requeue,
			"took", time.Since(begin),
			"err", err)
	}(time.Now())
	return r.Reconciler.Reconcile(kluster)
}
