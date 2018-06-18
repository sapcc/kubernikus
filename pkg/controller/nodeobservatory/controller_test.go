package nodeobservatory

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
)

const (
	KlusterName      = "fakeKluster"
	KlusterNamespace = "default"
	NodeName         = "node0"
)

func createFakeNodeObservatory(kluster *v1.Kluster, node *api_v1.Node) NodeObservatory {

	fakeKubernetesClientset := fake.NewSimpleClientset(node)
	fakeKubernikusClientset := kubernikusfake.NewSimpleClientset(kluster)
	kubernikusInformerFactory := kubernikus_informers.NewSharedInformerFactory(fakeKubernikusClientset, 0)

	return NodeObservatory{
		klusterInformer: kubernikusInformerFactory.Kubernikus().V1().Klusters(),
		clientFactory:   &kubernetes.MockSharedClientFactory{Clientset: fakeKubernetesClientset},
		logger:          log.NewNopLogger(),
	}
}

func TestReconilation(t *testing.T) {
	kluster := &v1.Kluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: KlusterNamespace,
			Name:      KlusterName,
		},
		Status: models.KlusterStatus{
			Phase: models.KlusterPhaseRunning,
		},
	}

	node := &api_v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: NodeName,
		},
		Status: api_v1.NodeStatus{
			Phase: api_v1.NodeRunning,
		},
	}

	no := createFakeNodeObservatory(kluster, node)
	require.NoError(t, no.reconcile(kluster))

	lister, err := no.GetListerForKluster(kluster)
	require.NoError(t, err)
	nodes, err := lister.List(labels.Everything())
	require.NoError(t, err)
	assert.Contains(t, nodes, node)

}
