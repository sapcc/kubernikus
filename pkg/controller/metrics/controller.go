package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(
		ControllerOperationsLatency,
		ControllerOperationsTotal,
		ControllerSuccessfulOperationsTotal,
		ControllerFailedOperationsTotal,
	)
}

var ControllerOperationsLatency = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "kubernikus",
		Subsystem: "controller",
		Name:      "operation_latency_seconds",
		Help:      "Total duration of reconciliation in microseconds.",
	},
	[]string{"controller", "method"})

var ControllerOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "controller",
		Name:      "operation_total",
		Help:      "Number of operations."},
	[]string{"controller", "method"})

var ControllerSuccessfulOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "controller",
		Name:      "successful_operation_total",
		Help:      "Number of successful operations."},
	[]string{"controller", "method"})

var ControllerFailedOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "controller",
		Name:      "failed_operation_total",
		Help:      "Number of failed operations."},
	[]string{"controller", "method"})
