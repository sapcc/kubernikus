package metrics

import "github.com/prometheus/client_golang/prometheus"

var DeorbitOperationsLatency = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "kubernikus",
		Subsystem: "deorbit",
		Name:      "operation_latency_seconds",
		Help:      "Total duration of reconciliation in microseconds.",
	},
	[]string{"method"})

var DeorbitOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "deorbit",
		Name:      "operation_total",
		Help:      "Number of operations."},
	[]string{"method"})

var DeorbitSuccessfulOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "deorbit",
		Name:      "successful_operation_total",
		Help:      "Number of successful operations."},
	[]string{"method"})

var DeorbitFailedOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "deorbit",
		Name:      "failed_operation_total",
		Help:      "Number of failed operations."},
	[]string{"method"})
