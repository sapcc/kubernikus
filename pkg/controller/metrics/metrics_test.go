package metrics

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

func TestMetrics(t *testing.T) {

	expectedMetrics := map[prometheus.Collector]string{
		nodePoolSize: `
# HELP kubernikus_node_pool_size size of a node pool
# TYPE kubernikus_node_pool_size gauge
kubernikus_node_pool_size{flavor_name="flavorName",image_name="imageName",kluster_id="klusterID",node_pool="nodePoolName"} 3
		`,
		nodePoolStatus: `
# HELP kubernikus_node_pool_status status of the node pool and the number of nodes nodes in that status
# TYPE kubernikus_node_pool_status gauge
kubernikus_node_pool_status{kluster_id="klusterID",node_pool="nodePoolName",status="ready"} 2
kubernikus_node_pool_status{kluster_id="klusterID",node_pool="nodePoolName",status="running"} 2
kubernikus_node_pool_status{kluster_id="klusterID",node_pool="nodePoolName",status="starting"} 1
		`,
		klusterInfo: `
# HELP kubernikus_kluster_info detailed information on a kluster
# TYPE kubernikus_kluster_info gauge
kubernikus_kluster_info{account="account",creator="D012345",kluster_name="klusterName",kluster_namespace="namespace",kluster_version="version",project_id="projectID"} 1
		`,
		klusterStatusPhase: `
# HELP kubernikus_kluster_status_phase the phase the kluster is currently in
# TYPE kubernikus_kluster_status_phase gauge
kubernikus_kluster_status_phase{kluster_id="klusterID",phase="Pending"} 0
kubernikus_kluster_status_phase{kluster_id="klusterID",phase="Creating"} 0
kubernikus_kluster_status_phase{kluster_id="klusterID",phase="Running"} 1
kubernikus_kluster_status_phase{kluster_id="klusterID",phase="Terminating"} 0
		`,
	}

	// call functions that update the metrics here
	setMetricNodePoolSize("klusterID", "nodePoolName", "imageName", "flavorName", 3)
	setMetricNodePoolStatus("klusterID", "nodePoolName", map[string]int64{"running": 2, "starting": 1, "ready": 2})
	SetMetricKlusterInfo("namespace", "klusterName", "version", "projectID", map[string]string{"creator": "D012345"}, map[string]string{"account": "account"})
	SetMetricKlusterStatusPhase("klusterID", models.KlusterPhaseRunning)

	registry := prometheus.NewPedanticRegistry()
	for collector, expectedMetricString := range expectedMetrics {
		// register the metric we're checking right now
		registry.MustRegister(collector)

		// collect aka gather
		actualMetrics, err := registry.Gather()
		if err != nil {
			t.Errorf("could not gather metrics: %#v", err)
		}
		// the actual check
		assert.NoError(t, compareMetrics(expectedMetricString, actualMetrics))

		// unregister to make sure we only have the metric we're checking right now
		if !registry.Unregister(collector) {
			t.Errorf("could not unregister %#v", collector)
		}
	}
}

// compare and return human readable error in case it's not equal
func compareMetrics(expectedMetrics string, actualMetrics []*dto.MetricFamily) error {
	var tp expfmt.TextParser
	expected, err := tp.TextToMetricFamilies(bytes.NewReader([]byte(expectedMetrics)))
	if err != nil {
		return fmt.Errorf("parsing expected metrics failed: %s", err)
	}

	if !reflect.DeepEqual(actualMetrics, normalizeMetricFamilies(expected)) {
		var buf1 bytes.Buffer
		enc := expfmt.NewEncoder(&buf1, expfmt.FmtText)
		for _, mf := range actualMetrics {
			if err := enc.Encode(mf); err != nil {
				return fmt.Errorf("encoding failed: %s", err)
			}
		}
		var buf2 bytes.Buffer
		enc = expfmt.NewEncoder(&buf2, expfmt.FmtText)
		for _, mf := range normalizeMetricFamilies(expected) {
			if err := enc.Encode(mf); err != nil {
				return fmt.Errorf("encoding failed: %s", err)
			}
		}

		return fmt.Errorf(`
unequal metric output;
want:

%s

got:

%s
`, buf2.String(), buf1.String())
	}
	return nil
}

func normalizeMetricFamilies(metricFamiliesByName map[string]*dto.MetricFamily) []*dto.MetricFamily {
	for _, mf := range metricFamiliesByName {
		sort.Sort(metricSorter(mf.Metric))
	}
	names := make([]string, 0, len(metricFamiliesByName))
	for name, mf := range metricFamiliesByName {
		if len(mf.Metric) > 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := make([]*dto.MetricFamily, 0, len(names))
	for _, name := range names {
		result = append(result, metricFamiliesByName[name])
	}
	return result
}

type metricSorter []*dto.Metric

func (s metricSorter) Len() int {
	return len(s)
}

func (s metricSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s metricSorter) Less(i, j int) bool {
	sort.Sort(prometheus.LabelPairSorter(s[i].Label))
	sort.Sort(prometheus.LabelPairSorter(s[j].Label))

	if len(s[i].Label) != len(s[j].Label) {
		return len(s[i].Label) < len(s[j].Label)
	}

	for n, lp := range s[i].Label {
		vi := lp.GetValue()
		vj := s[j].Label[n].GetValue()
		if vi != vj {
			return vi < vj
		}
	}

	if s[i].TimestampMs == nil {
		return false
	}
	if s[j].TimestampMs == nil {
		return true
	}
	return s[i].GetTimestampMs() < s[j].GetTimestampMs()
}
