package metrics

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/golang/glog"
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
	models.KlusterPhaseTerminating,
}

var klusterInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kluster_info",
		Help:      "detailed information on a kluster",
	},
	[]string{"kluster_namespace", "kluster_name", "kluster_version", "creator", "account", "project_id"},
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

func SetMetricKlusterInfo(namespace, name, version, projectID string, annotations, labels map[string]string) {
	promLabels := prometheus.Labels{
		"kluster_namespace": namespace,
		"kluster_name":      name,
		"kluster_version":   version,
		"creator":           getCreatorFromAnnotations(annotations),
		"account":           getAccountFromLabels(labels),
		"project_id":        projectID,
	}
	klusterInfo.With(promLabels).Set(1)
}

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

/*
kubernikus_node_pool_size{"kluster_id"="<id", "node_pool"="<name>", "image_name"="<name>", "flavor_name"="<name>"} <node_pool_size>
*/
func setMetricNodePoolSize(klusterID, nodePoolName, imageName, flavorName string, size int64) {
	nodePoolSize.With(prometheus.Labels{
		"kluster_id":  klusterID,
		"node_pool":   nodePoolName,
		"image_name":  imageName,
		"flavor_name": flavorName,
	}).Set(float64(size))
}

/*
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="<status>"} < number of nodes in that status >
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="ready"} 1
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="running"} 1
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="healthy"} 1
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="error"} 1
*/
func setMetricNodePoolStatus(klusterID, nodePoolName string, status map[string]int64) {
	if status != nil {
		for s, v := range status {
			nodePoolStatus.With(prometheus.Labels{
				"kluster_id": klusterID,
				"node_pool":  nodePoolName,
				"status":     s,
			}).Set(float64(v))
		}
	}
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

func init() {
	prometheus.MustRegister(
		klusterInfo,
		klusterStatusPhase,
		nodePoolSize,
		nodePoolStatus,
		LaunchOperationsLatency,
		LaunchOperationsTotal,
		LaunchSuccessfulOperationsTotal,
		LaunchFailedOperationsTotal,
	)
}

func ExposeMetrics(metricPort int, stopCh <-chan struct{}, wg *sync.WaitGroup) error {
	glog.Infof("Exposing metrics on localhost:%v/metrics ", metricPort)
	defer wg.Done()
	wg.Add(1)
	for {
		select {
		case <-stopCh:
			return nil
		default:
			http.Handle("/metrics", promhttp.Handler())
			return http.ListenAndServe(
				fmt.Sprintf("0.0.0.0:%v", metricPort),
				nil,
			)
		}
	}
}
