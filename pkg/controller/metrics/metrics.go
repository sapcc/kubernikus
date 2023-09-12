package metrics

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

const (
	metricNamespace = "kubernikus"
)

var klusterPhases = []models.KlusterPhase{
	models.KlusterPhasePending,
	models.KlusterPhaseCreating,
	models.KlusterPhaseRunning,
	models.KlusterPhaseUpgrading,
	models.KlusterPhaseTerminating,
}

var klusterBootDurationSummary = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: metricNamespace,
		Name:      "kluster_boot_duration",
		Help:      "Duration until kluster got from phase pending to running",
	},
	[]string{},
)

var klusterStatusPhase = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kluster_status_phase",
		Help:      "the phase the kluster is currently in",
	},
	[]string{"kluster_id", "phase"},
)

var nodePoolSize = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_pool_size",
		Help:      "size of a node pool",
	},
	[]string{"kluster_id", "node_pool", "image_name", "flavor_name"},
)

var nodePoolStatus = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_pool_status",
		Help:      "status of the node pool and the number of nodes nodes in that status",
	},
	[]string{"kluster_id", "node_pool", "status"},
)

/*
kubernikus_kluster_status_phase{"kluster_id"="<id>","phase"="<phase>"} 			< 1|0 >
kubernikus_kluster_status_phase{"kluster_id"="<id>","phase"="creating"} 		1
kubernikus_kluster_status_phase{"kluster_id"="<id>","phase"="running"} 			0
kubernikus_kluster_status_phase{"kluster_id"="<id>","phase"="pending"} 			0
kubernikus_kluster_status_phase{"kluster_id"="<id>","phase"="terminating"} 	0
*/
func SetMetricKlusterStatusPhase(klusterName string, klusterPhase models.KlusterPhase) {
	// Set current phase to 1, others to 0
	for _, phase := range klusterPhases {
		labels := prometheus.Labels{
			"kluster_id": klusterName,
			"phase":      string(phase),
		}
		klusterStatusPhase.With(labels).Set(boolToFloat64(klusterPhase == phase))
	}
}

func SetMetricKlusterTerminated(klusterName string) {
	for _, phase := range klusterPhases {
		labels := prometheus.Labels{
			"kluster_id": klusterName,
			"phase":      string(phase),
		}
		klusterStatusPhase.With(labels).Set(0)
	}
}

/*
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="<status>"} < number of nodes in that status >
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="schedulable"} 1
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="running"} 1
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="healthy"} 1
*/
func SetMetricNodePoolStatus(klusterID, nodePoolName string, status map[string]int64) {
	for s, v := range status {
		nodePoolStatus.With(prometheus.Labels{
			"kluster_id": klusterID,
			"node_pool":  nodePoolName,
			"status":     s,
		}).Set(float64(v))
	}
}

func SetMetricBootDurationSummary(creationTimestamp, now time.Time) {
	klusterBootDurationSummary.With(prometheus.Labels{}).Observe(now.Sub(creationTimestamp).Seconds())
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func getCreatorFromAnnotations(annotations map[string]string) string {
	creator, ok := annotations["creator"]
	if !ok {
		return "NA"
	}
	return creator
}

func getAccountFromLabels(labels map[string]string) string {
	account, ok := labels["account"]
	if !ok {
		return "NA"
	}
	return account
}

func getAuditFromSpec(spec models.KlusterSpec) string {
	if spec.Audit == nil {
		return ""
	}
	return *spec.Audit
}

func getBackupFromSpec(spec models.KlusterSpec) string {
	switch spec.Backup {
	case "", "on":
		return "swift"
	default:
		return spec.Backup
	}
}

func init() {
	prometheus.MustRegister(
		klusterStatusPhase,
		klusterBootDurationSummary,
		nodePoolSize,
		nodePoolStatus,
	)
}

func ExposeMetrics(host string, metricPort int, stopCh <-chan struct{}, wg *sync.WaitGroup, logger log.Logger) {
	wg.Add(1)
	defer wg.Done()
	ln, err := net.Listen("tcp", fmt.Sprintf("%v:%v", host, metricPort))
	logger.Log(
		"msg", "Exposing metrics",
		"host", host,
		"port", metricPort,
		"err", err)
	if err != nil {
		return
	}
	go http.Serve(ln, promhttp.Handler())
	<-stopCh
	ln.Close()
}
