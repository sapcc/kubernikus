package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(
		FlightFailedSecurityGroupOperationsTotal,
	)
}

var FlightFailedSecurityGroupOperationsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "flight",
		Name:      "security_group_rules_errors_total",
		Help:      "Total number of error while ensuring security group rules",
	},
	[]string{"kluster"})
