package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/labels"

	kubernikus_lister "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
)

var SeedReconciliationFailuresTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "kubernikus",
		Subsystem: "seed_reconciliation",
		Name:      "failures_total",
		Help:      "Number of failed seed reconciliations."},
	[]string{"kluster_name"})

type klusterCollector struct {
	infoMetric *prometheus.Desc
	lister     kubernikus_lister.KlusterLister
}

func RegisterKlusterCollector(lister kubernikus_lister.KlusterLister) {
	prometheus.MustRegister(NewKlusterCollector(lister))
}

func NewKlusterCollector(lister kubernikus_lister.KlusterLister) *klusterCollector {
	return &klusterCollector{
		infoMetric: prometheus.NewDesc(
			"kubernikus_kluster_info",
			"Detailed information on a kluster",
			[]string{"kluster_namespace", "kluster_name", "phase", "api_version", "chart_version", "chart_name", "creator", "project_id", "backup", "audit"},
			nil,
		),
		lister: lister,
	}
}

// Each and every collector must implement the Describe function.
// It essentially writes all descriptors to the prometheus desc channel.
func (collector *klusterCollector) Describe(ch chan<- *prometheus.Desc) {
	//Update this section with the each metric you create for a given collector
	ch <- collector.infoMetric
}

// Collect implements required collect function for all promehteus collectors
func (collector *klusterCollector) Collect(ch chan<- prometheus.Metric) {
	klusters, err := collector.lister.List(labels.Everything())
	if err != nil {
		return
	}

	var value float64 = 1

	for _, kluster := range klusters {
		ch <- prometheus.MustNewConstMetric(
			collector.infoMetric,
			prometheus.GaugeValue,
			value,
			kluster.GetNamespace(),
			kluster.GetName(),
			string(kluster.Status.Phase),
			kluster.Status.ApiserverVersion,
			kluster.Status.ChartVersion,
			kluster.Status.ChartName,
			getCreatorFromAnnotations(kluster.Annotations),
			getAccountFromLabels(kluster.Labels),
			kluster.Spec.Backup,
			*kluster.Spec.Audit,
		)
	}

	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	//ch <- prometheus.MustNewConstMetric(collector.fooMetric, prometheus.CounterValue, metricValue)

}
