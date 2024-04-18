package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	backendGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: Prefix + "configured_backends_count",
			Help: "Number of configured backends",
		},
	)
)

func SetBackendCount(count int) {
	backendGauge.Set(float64(count))
}
