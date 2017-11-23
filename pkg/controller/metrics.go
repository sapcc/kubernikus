package controller

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

var klusterInstancesTotal = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kluster_instances_total",
		Help:      "total number of klusters",
	},
	[]string{"domain_id", "project_id"},
)

var klusterStatusPhase = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kluster_status_phase",
		Help:      "The phase the kluster is currently in",
	},
	[]string{"kluster_id", "phase"},
)

var nodePoolInfo = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_pool_info",
		Help:      "information for a node pool",
	},
	[]string{"kluster_id", "node_pool", "image_name", "flavor_name"},
)

var nodePoolStatus = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_pool_status",
		Help:      "status of the node pool",
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
func setMetricStatusPhaseForKluster(klusterName string, klusterPhase models.KlusterPhase) {
	// Set current phase to 1, others to 0 if it is set.
	if klusterPhase != "" {
		labels := prometheus.Labels{
			"kluster_id": klusterName,
			"phase":      string(klusterPhase),
		}
		klusterStatusPhase.With(labels).Set(boolToFloat64(klusterPhase == models.KlusterPhaseCreating))
		klusterStatusPhase.With(labels).Set(boolToFloat64(klusterPhase == models.KlusterPhaseRunning))
		klusterStatusPhase.With(labels).Set(boolToFloat64(klusterPhase == models.KlusterPhasePending))
		klusterStatusPhase.With(labels).Set(boolToFloat64(klusterPhase == models.KlusterPhaseTerminating))
	}
}

/*
kubernikus_node_pool_info{"kluster_id"="<id", "node_pool"="<name>", "image_name"="<name>", "flavor_name"="<name>"} <node_pool_size>
*/
func setMetricNodePoolSize(klusterID, nodePoolName, imageName, flavorName string, nodePoolSize int64) {
	nodePoolInfo.With(prometheus.Labels{
		"kluster_id":  klusterID,
		"node_pool":   nodePoolName,
		"image_name":  imageName,
		"flavor_name": flavorName,
	}).Set(float64(nodePoolSize))
}

/*
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="<status>"} < number of nodes in that status >
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="ready"} 1
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="running"} 1
kubernikus_node_pool_status{"kluster_id"="<id", "node_pool"="<name>", "status"="healthy"} 1
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

func init() {
	prometheus.MustRegister(
		klusterInstancesTotal,
		klusterStatusPhase,
		nodePoolInfo,
		nodePoolStatus,
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
