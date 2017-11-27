package metrics

import "github.com/prometheus/client_golang/prometheus"

var KlusterReconcilicationCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "launch",
		Name:      "reconciliation_count",
		Help:      "Number of reconcilitations."},
	[]string{"kluster", "project"})

var KlusterReconciliationLatency = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "kubernikus",
		Subsystem: "launch",
		Name:      "reconciliation_latency_microseconds",
		Help:      "Total duration of reconciliation in microseconds.",
	},
	[]string{"kluster", "project"})
