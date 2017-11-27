package base

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type InstrumentingReconciler struct {
	Reconciler
	ReconciliationCount   *prometheus.CounterVec
	ReconciliationLatency *prometheus.HistogramVec
}

func (ir *InstrumentingReconciler) Reconcile(kluster *v1.Kluster) (requeue bool, err error) {
	defer func(begin time.Time) {
		ir.ReconciliationCount.With(
			prometheus.Labels{
				"kluster": kluster.Spec.Name,
				"project": kluster.Account(),
			}).Add(1)
		ir.ReconciliationLatency.With(
			prometheus.Labels{
				"kluster": kluster.Spec.Name,
				"project": kluster.Account(),
			}).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ir.Reconciler.Reconcile(kluster)
}
