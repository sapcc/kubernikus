package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(
		MigrationErrorsTotal,
	)
}

var MigrationErrorsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "migration",
		Name:      "errors_total",
		Help:      "Total numver of failed migration operations",
	},
	[]string{"kluster"},
)
