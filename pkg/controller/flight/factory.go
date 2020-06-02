package flight

import (
	"github.com/go-kit/kit/log"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	openstack_kluster "github.com/sapcc/kubernikus/pkg/client/openstack/kluster"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
)

type FlightReconcilerFactory interface {
	FlightReconciler(*v1.Kluster) (FlightReconciler, error)
}

type flightReconcilerFactory struct {
	Openstack        openstack.SharedOpenstackClientFactory
	KubernetesClient kubernetes.Interface
	NodeObservatory  *nodeobservatory.NodeObservatory
	Recorder         record.EventRecorder
	Logger           log.Logger
}

func NewFlightReconcilerFactory(openstack openstack.SharedOpenstackClientFactory, kubernetesClient kubernetes.Interface, nodeObservatory *nodeobservatory.NodeObservatory, recorder record.EventRecorder, logger log.Logger) FlightReconcilerFactory {
	return &flightReconcilerFactory{openstack, kubernetesClient, nodeObservatory, recorder, logger}
}

func (f *flightReconcilerFactory) FlightReconciler(kluster *v1.Kluster) (FlightReconciler, error) {
	client, err := f.Openstack.KlusterClientFor(kluster)
	if err != nil {
		return nil, err
	}

	adminClient, err := f.Openstack.AdminClient()
	if err != nil {
		return nil, err
	}

	instances, err := f.getInstances(kluster, client)
	if err != nil {
		return nil, err
	}

	lister, err := f.NodeObservatory.GetListerForKluster(kluster)
	if err != nil {
		return nil, err
	}

	nodes, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var reconciler FlightReconciler
	reconciler = &flightReconciler{kluster, instances, nodes, client, f.KubernetesClient, adminClient, f.Logger}
	reconciler = &LoggingFlightReconciler{reconciler, f.Logger}
	return reconciler, nil
}

func (d *flightReconcilerFactory) getInstances(kluster *v1.Kluster, client openstack_kluster.KlusterClient) ([]Instance, error) {
	instances := []Instance{}
	for _, pool := range kluster.Spec.NodePools {
		if poolNodes, err := client.ListNodes(kluster, &pool); err != nil {
			return nil, err
		} else {
			for _, n := range poolNodes {
				n := n
				instances = append(instances, &n)
			}
		}
	}

	return instances, nil
}
