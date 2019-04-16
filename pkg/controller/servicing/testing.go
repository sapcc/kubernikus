package servicing

import (
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/go-kit/kit/log"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
	informers_kubernikus "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
	listers_kubernikus_v1 "github.com/sapcc/kubernikus/pkg/generated/listers/kubernikus/v1"
)

var (
	// TestDebugEnabled can be used to turn on debug logging for tests
	TestDebugEnabled = func() bool { return false }

	// TestLogger creates a Logger for tests
	TestLogger = func() log.Logger {
		if TestDebugEnabled() {
			return log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
		}
		return log.NewNopLogger()
	}
)

// FakeKlusterOptions are used to describe a Kluster's properties for tests
type FakeKlusterOptions struct {
	Phase       models.KlusterPhase
	NodePools   []FakeNodePoolOptions
	LastService *time.Time
}

// FakeNodePoolOptions are used to describe a Nodepool for tests
type FakeNodePoolOptions struct {
	AllowReboot         bool
	AllowReplace        bool
	NodeHealthy         bool
	NodeOSOutdated      bool
	NodeKubeletOutdated bool
	NodeUpdating        *time.Time
	Size                int
}

// NewFakeKluster creates a Kluster Object for tests
func NewFakeKluster(opts *FakeKlusterOptions) (*v1.Kluster, []runtime.Object) {
	kluster := &v1.Kluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "servicing",
			Name:      "test",
			Annotations: map[string]string{
				AnnotationServicingSafeguard: "true",
			},
		},
		Spec: models.KlusterSpec{
			Name:      "test",
			NodePools: []models.NodePool{},
		},
		Status: models.KlusterStatus{
			Phase:            opts.Phase,
			ApiserverVersion: "v1.10.15",
		},
	}

	nodes := []runtime.Object{}

	for i, p := range opts.NodePools {
		poolName := fmt.Sprintf("pool%d", i)
		pool := models.NodePool{
			Name: poolName,
			Config: &models.NodePoolConfig{
				AllowReplace: p.AllowReboot,
				AllowReboot:  p.AllowReplace,
			},
		}
		kluster.Spec.NodePools = append(kluster.Spec.NodePools, pool)

		for j := 0; j < p.Size; j++ {
			nodeName := fmt.Sprintf("test-%s-0000%d", poolName, j)
			node := &core_v1.Node{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:        nodeName,
					Annotations: map[string]string{},
				},
				Status: core_v1.NodeStatus{
					Phase:    core_v1.NodeRunning,
					NodeInfo: core_v1.NodeSystemInfo{},
					Conditions: []core_v1.NodeCondition{
						{
							Type:   core_v1.NodeReady,
							Status: core_v1.ConditionUnknown,
						},
					},
				},
			}

			if p.NodeUpdating != nil {
				node.ObjectMeta.Annotations[AnnotationUpdateTimestamp] = p.NodeUpdating.UTC().Format(time.RFC3339)
			}

			if p.NodeHealthy {
				node.Status.Conditions[0].Status = core_v1.ConditionTrue
			} else {
				node.Status.Conditions[0].Status = core_v1.ConditionFalse
			}

			if p.NodeOSOutdated {
				node.Status.NodeInfo.OSImage = "Container Linux by CoreOS 1800.6.0 (Rhyolite)"
			} else {
				node.Status.NodeInfo.OSImage = "Container Linux by CoreOS 2023.4.0 (Rhyolite)"
			}

			if p.NodeKubeletOutdated {
				node.Status.NodeInfo.KubeletVersion = "v1.10.11"
				node.Status.NodeInfo.KubeProxyVersion = "v1.10.11"
			} else {
				node.Status.NodeInfo.KubeletVersion = "v1.10.15"
				node.Status.NodeInfo.KubeProxyVersion = "v1.10.15"
			}

			nodes = append(nodes, node)
		}
	}

	if opts.LastService != nil {
		kluster.Annotations[AnnotationServicingTimestamp] = (*opts.LastService).Format(time.RFC3339)
	}

	return kluster, nodes
}

// NewFakeKlusterLister creates a Fake Lister for tests
func NewFakeKlusterLister(k *v1.Kluster) listers_kubernikus_v1.KlusterLister {
	fakeClientset := kubernikusfake.NewSimpleClientset(k)
	fakeFactory := informers_kubernikus.NewSharedInformerFactory(fakeClientset, 0)
	fakeFactory.Kubernikus().V1().Klusters().Informer()
	fakeFactory.Start(wait.NeverStop)
	fakeFactory.WaitForCacheSync(wait.NeverStop)

	return fakeFactory.Kubernikus().V1().Klusters().Lister()
}
