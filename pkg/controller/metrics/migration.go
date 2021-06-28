package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(
		MigrationErrorsTotal,
	)
	MigrationErrorsTotal.With(prometheus.Labels{"kluster": "dummy-for-absent-metrics-operator"}).Add(0)
}

var MigrationErrorsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "migration",
		Name:      "errors_total",
		Help:      "Total number of failed migration operations",
	},
	[]string{"kluster"},
)
