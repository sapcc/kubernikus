package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(
		HammertimeStatus,
	)
}

var HammertimeStatus = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "kubernikus",
		Subsystem: "hammertime",
		Name:      "status",
		Help:      "Status of hammertime (controler manager scaled down)",
	},
	[]string{"kluster"},
)
