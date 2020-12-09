package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(
		LaunchOperationsLatency,
		LaunchOperationsTotal,
		LaunchSuccessfulOperationsTotal,
		LaunchFailedOperationsTotal,
	)
	//Make sure absent metrics operator is happy:
	LaunchOperationsLatency.With(prometheus.Labels{"method": "GetStatus"}).Observe(0)
	LaunchOperationsTotal.With(prometheus.Labels{"method": "GetStatus"}).Add(0)
	LaunchSuccessfulOperationsTotal.With(prometheus.Labels{"method": "GetStatus"}).Add(0)
	LaunchFailedOperationsTotal.With(prometheus.Labels{"method": "GetStatus"}).Add(0)
}

var LaunchOperationsLatency = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "kubernikus",
		Subsystem: "launch",
		Name:      "operation_latency_seconds",
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
