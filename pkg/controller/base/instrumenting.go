package base

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type InstrumentingReconciler struct {
	Reconciler

	Latency    *prometheus.SummaryVec
	Total      *prometheus.CounterVec
	Successful *prometheus.CounterVec
	Failed     *prometheus.CounterVec
}

func (ir *InstrumentingReconciler) Reconcile(kluster *v1.Kluster) (requeue bool, err error) {
	defer func(begin time.Time) {
		ir.Latency.With(
			prometheus.Labels{
				"method": "Reconcile",
			}).Observe(time.Since(begin).Seconds())

		ir.Total.With(
			prometheus.Labels{
				"method": "Reconcile",
			}).Add(1)

		if err != nil {
			ir.Failed.With(
				prometheus.Labels{
					"method": "Reconcile",
				}).Add(1)
		} else {
			ir.Successful.With(
				prometheus.Labels{
					"method": "Reconcile",
				}).Add(1)
		}
	}(time.Now())
	return ir.Reconciler.Reconcile(kluster)
}
