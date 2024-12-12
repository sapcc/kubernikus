package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	connDurationsHisto = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: Prefix + "connection_durations_histogram_seconds",
		Help: "Connection duration distributions.",
	})
	connGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: Prefix + "opened_connections_count",
			Help: "Number of opened TCP connections",
		},
	)
)

// IncConnections increments the total connections counter
func IncConnections() {
	connGauge.Inc()
}

func DecConnections() {
	connGauge.Dec()
}

// ConnectionTime gather the duration of a connection
func ConnectionTime(d time.Duration) {
	connDurationsHisto.Observe(d.Seconds())
}
