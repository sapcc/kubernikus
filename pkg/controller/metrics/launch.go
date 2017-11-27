package metrics

import "github.com/prometheus/client_golang/prometheus"

var LaunchOperationsLatency = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "kubernikus",
		Subsystem: "launch",
		Name:      "operation_latency_microseconds",
		Help:      "Total duration of reconciliation in microseconds.",
	},
	[]string{"method"})

var LaunchOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "launch",
		Name:      "operation_total",
		Help:      "Number of operations."},
	[]string{"method"})

var LaunchSuccessfulOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "launch",
		Name:      "successful_operation_total",
		Help:      "Number of successful operations."},
	[]string{"method"})

var LaunchFailedOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "launch",
		Name:      "failed_operation_total",
		Help:      "Number of failed operations."},
	[]string{"method"})
