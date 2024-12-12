package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	Error = "error"
	Fatal = "fatal"
	Info  = "info"
)

var (
	errorCounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: Prefix + "errors_total",
			Help: "Total error count",
		},
		[]string{"type"},
	)
)

// IncErrors increments the total errors counter
func IncErrors(typ string) {
	errorCounterVec.WithLabelValues(typ).Inc()
}
