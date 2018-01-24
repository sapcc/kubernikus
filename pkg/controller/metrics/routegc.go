package metrics

import "github.com/prometheus/client_golang/prometheus"

var OrphanedRoutesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "routegc",
		Name:      "orphaned_routes_total",
		Help:      "Number of orphaned routes removed from OpenStack router",
	},
	[]string{},
)

var RouteGCFailedOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "routgc",
		Name:      "failed_operation_total",
		Help:      "Number of failed operations.",
	},
	[]string{},
)
