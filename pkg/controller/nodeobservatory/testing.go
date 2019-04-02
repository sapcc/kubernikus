package nodeobservatory

import (
	"github.com/go-kit/kit/log"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetes_fake "k8s.io/client-go/kubernetes/fake"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	kubernikus_fake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
	kubernikus_informers "github.com/sapcc/kubernikus/pkg/generated/informers/externalversions"
)

func NewFakeController(kluster *v1.Kluster, nodes ...runtime.Object) *NodeObservatory {
	fakeKubernetesClientset := kubernetes_fake.NewSimpleClientset(nodes...)
	fakeKubernikusClientset := kubernikus_fake.NewSimpleClientset(kluster)
	kubernikusInformerFactory := kubernikus_informers.NewSharedInformerFactory(fakeKubernikusClientset, 0)

	controller := &NodeObservatory{
		klusterInformer: kubernikusInformerFactory.Kubernikus().V1().Klusters(),
		clientFactory:   &kube.MockSharedClientFactory{Clientset: fakeKubernetesClientset},
		logger:          log.NewNopLogger(),
	}

	controller.createAndWatchNodeInformerForKluster(kluster)

	return controller
}
