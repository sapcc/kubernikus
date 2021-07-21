package servicing

import (
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/coreos"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/flatcar"
	kubernikus_lister "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
)

type servicingNodesCollector struct {
	klusters    kubernikus_lister.KlusterLister
	nodeListers *NodeListerFactory
	updating    *prometheus.Desc
	waiting     *prometheus.Desc
	kubelet     *prometheus.Desc
	proxy       *prometheus.Desc
	osimage     *prometheus.Desc
	logger      log.Logger
}

// RegisterServicingNodesCollector does what the method name sais
func RegisterServicingNodesCollector(logger log.Logger, factories config.Factories) {
	collector := &servicingNodesCollector{
		updating: prometheus.NewDesc(
			"kubernikus_servicing_nodes_state_updating",
			"Amount of nodes per servicing action/state",
			[]string{"kluster_id", "state"},
			prometheus.Labels{},
		),
		waiting: prometheus.NewDesc(
			"kubernikus_servicing_nodes_state_waiting",
			"Amount of nodes that are waiting for updates per state",
			[]string{"kluster_id", "state"},
			prometheus.Labels{},
		),
		kubelet: prometheus.NewDesc(
			"kubernikus_servicing_nodes_version_kubelet",
			"Update Status of Nodes per Kluster",
			[]string{"kluster_id", "version"},
			prometheus.Labels{},
		),
		proxy: prometheus.NewDesc(
			"kubernikus_servicing_nodes_version_proxy",
			"Update Status of Nodes per Kluster",
			[]string{"kluster_id", "version"},
			prometheus.Labels{},
		),
		osimage: prometheus.NewDesc(
			"kubernikus_servicing_nodes_version_osimage",
			"Update Status of Nodes per Kluster",
			[]string{"kluster_id", "version"},
			prometheus.Labels{},
		),
		klusters: factories.Kubernikus.Kubernikus().V1().Klusters().Lister(),
		nodeListers: &NodeListerFactory{
			Logger:          logger,
			NodeObservatory: factories.NodesObservatory.NodeInformer(),
			CoreOSVersion:   &coreos.Version{},
			CoreOSRelease:   &coreos.Release{},
			FlatcarVersion:  &flatcar.Version{},
			FlatcarRelease:  &flatcar.Release{},
		},
		logger: logger,
	}

	prometheus.MustRegister(collector)
}

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel.
func (c *servicingNodesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.updating
	ch <- c.waiting
	ch <- c.kubelet
	ch <- c.proxy
	ch <- c.osimage
}

//Collect implements required collect function for all promehteus collectors
func (c *servicingNodesCollector) Collect(ch chan<- prometheus.Metric) {
	klusters, err := c.klusters.List(labels.Everything())
	if err != nil {
		c.logger.Log("msg", "Failed to list klusters", "err", err)
		return
	}

	for _, kluster := range klusters {
		nodes, err := c.nodeListers.Make(kluster)
		if err != nil {
			c.logger.Log("msg", "Failed to list nodes", "kluster", kluster.Name, "err", err)
			continue
		}

		updatingStarted := float64(len(nodes.Updating()))
		updatingFailed := float64(len(nodes.Failed()))
		updatingSuccessful := float64(len(nodes.Successful()))

		waitingReboot := float64(len(nodes.Reboot()))
		waitingReplace := float64(len(nodes.Replace()))
		waitingUptodate := float64(len(nodes.All())) - waitingReboot - waitingReplace

		kubeletVersions := map[string]int{}
		proxyVersions := map[string]int{}
		osVersions := map[string]int{}

		for _, node := range nodes.All() {
			kubeletVersions[node.Status.NodeInfo.KubeletVersion]++
			proxyVersions[node.Status.NodeInfo.KubeProxyVersion]++

			osVersion, err := flatcar.ExractVersion(node)
			if err != nil {
				if osVersion, err = coreos.ExractVersion(node); err != nil {
					continue
				}
			}
			osVersions[osVersion.String()]++
		}

		ch <- prometheus.MustNewConstMetric(c.updating, prometheus.GaugeValue, updatingStarted, kluster.GetName(), "started")
		ch <- prometheus.MustNewConstMetric(c.updating, prometheus.GaugeValue, updatingFailed, kluster.GetName(), "failed")
		ch <- prometheus.MustNewConstMetric(c.updating, prometheus.GaugeValue, updatingSuccessful, kluster.GetName(), "successful")
		ch <- prometheus.MustNewConstMetric(c.waiting, prometheus.GaugeValue, waitingReboot, kluster.GetName(), "reboot")
		ch <- prometheus.MustNewConstMetric(c.waiting, prometheus.GaugeValue, waitingReplace, kluster.GetName(), "replace")
		ch <- prometheus.MustNewConstMetric(c.waiting, prometheus.GaugeValue, waitingUptodate, kluster.GetName(), "uptodate")

		for version, count := range kubeletVersions {
			ch <- prometheus.MustNewConstMetric(c.kubelet, prometheus.GaugeValue, float64(count), kluster.GetName(), version)
		}

		for version, count := range proxyVersions {
			ch <- prometheus.MustNewConstMetric(c.proxy, prometheus.GaugeValue, float64(count), kluster.GetName(), version)
		}

		for version, count := range osVersions {
			ch <- prometheus.MustNewConstMetric(c.osimage, prometheus.GaugeValue, float64(count), kluster.GetName(), version)
		}
	}
}
