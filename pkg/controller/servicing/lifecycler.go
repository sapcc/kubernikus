package servicing

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	kube "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
	openstack_kluster "github.com/sapcc/kubernikus/pkg/client/openstack/kluster"
	"github.com/sapcc/kubernikus/pkg/controller/events"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/drain"
)

const (
	// EvictionTimeout defines when to abort the draining of a node
	EvictionTimeout = 10 * time.Minute
)

type (
	// LifeCycler managed a node's lifecycle actions
	LifeCycler interface {
		Drain(node *core_v1.Node) error
		Reboot(node *core_v1.Node) error
		Replace(node *core_v1.Node) error
	}

	// LifeCyclerFactory creates a LifeCycler for a Kluster
	LifeCyclerFactory interface {
		Make(*v1.Kluster) (LifeCycler, error)
	}

	// NodeLifeCyclerFactory produces LifeCyclers that can manage Openstack based nodes
	NodeLifeCyclerFactory struct {
		Recorder   record.EventRecorder
		Logger     log.Logger
		Satellites kube.SharedClientFactory
		Openstack  openstack.SharedOpenstackClientFactory
	}

	// NodeLifeCycler manages Openstack based Nodes
	NodeLifeCycler struct {
		Logger     log.Logger
		Kubernetes kubernetes.Interface
		Openstack  openstack_kluster.KlusterClient
	}

	// LoggingLifeCycler logs lifecycle actions
	LoggingLifeCycler struct {
		LifeCycler LifeCycler
		Logger     log.Logger
	}

	// EventingLifeCycler produces lifecycle events to be disabled for the end-user
	EventingLifeCycler struct {
		LifeCycler LifeCycler
		Kluster    *v1.Kluster
		Recorder   record.EventRecorder
	}

	// InstrumentingLifeCycler produces Prometheus metrics for Lifecycle actions
	InstrumentingLifeCycler struct {
		LifeCycler LifeCycler

		Latency    *prometheus.SummaryVec
		Total      *prometheus.CounterVec
		Successful *prometheus.CounterVec
		Failed     *prometheus.CounterVec
	}
)

// Make produces a LifeCycler for a specific Kluster
func (l *NodeLifeCyclerFactory) Make(k *v1.Kluster) (LifeCycler, error) {
	var lifeCycler LifeCycler
	logger := log.With(l.Logger, "kluster", k.Spec.Name, "project", k.Account())

	kubernetes, err := l.Satellites.ClientFor(k)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't create Kubernikus client")
	}

	openstack, err := l.Openstack.KlusterClientFor(k)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't create Openstack client")
	}

	lifeCycler = &NodeLifeCycler{
		Logger:     logger,
		Kubernetes: kubernetes,
		Openstack:  openstack,
	}

	lifeCycler = &LoggingLifeCycler{
		LifeCycler: lifeCycler,
		Logger:     logger,
	}

	lifeCycler = &EventingLifeCycler{
		LifeCycler: lifeCycler,
		Recorder:   l.Recorder,
		Kluster:    k,
	}

	lifeCycler = &InstrumentingLifeCycler{
		LifeCycler: lifeCycler,
		Latency:    metrics.ControllerOperationsLatency,
		Total:      metrics.ControllerOperationsTotal,
		Successful: metrics.ControllerSuccessfulOperationsTotal,
		Failed:     metrics.ControllerFailedOperationsTotal,
	}

	return lifeCycler, nil
}

// Drain uses a copy of openshift/kubernetes-drain to drain a node
// It is based on code extracted from kubectl, modified with kit-log
// compliant logging
func (lc *NodeLifeCycler) Drain(node *core_v1.Node) error {
	options := &drain.DrainOptions{
		Force:              false,
		IgnoreDaemonsets:   true,
		GracePeriodSeconds: -1,
		Timeout:            EvictionTimeout,
		DeleteLocalData:    false,
		Namespace:          meta_v1.NamespaceAll,
		Selector:           nil,
		Logger:             log.With(lc.Logger, "node", node.GetName()),
	}
	err := drain.Drain(lc.Kubernetes, []*core_v1.Node{node}, options)
	return err
}

// Reboot a node softly
func (lc *NodeLifeCycler) Reboot(node *core_v1.Node) error {
	if err := lc.Openstack.RebootNode(node.Spec.ExternalID); err != nil {
		return errors.Wrap(err, "rebooting node failed")
	}
	return nil
}

// Replace a node by temrinating it
func (lc *NodeLifeCycler) Replace(node *core_v1.Node) error {
	if err := lc.Openstack.DeleteNode(node.Spec.ExternalID); err != nil {
		return errors.Wrap(err, "deleting node failed")
	}
	return nil
}

// Drain logs the action
func (lc *LoggingLifeCycler) Drain(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		lc.Logger.Log(
			"msg", "draining node",
			"node", node.GetName(),
			"took", time.Since(begin),
			"v", 1,
			"err", err,
		)
	}(time.Now())
	return lc.LifeCycler.Drain(node)
}

// Reboot logs the action
func (lc *LoggingLifeCycler) Reboot(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		lc.Logger.Log(
			"msg", "rebooting node",
			"node", node.GetName(),
			"took", time.Since(begin),
			"v", 1,
			"err", err,
		)
	}(time.Now())
	return lc.LifeCycler.Reboot(node)
}

// Replace logs the action
func (lc *LoggingLifeCycler) Replace(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		lc.Logger.Log(
			"msg", "replacing node",
			"node", node.GetName(),
			"took", time.Since(begin),
			"v", 1,
			"err", err,
		)
	}(time.Now())
	return lc.LifeCycler.Replace(node)
}

// Drain writes an Event
func (lc *EventingLifeCycler) Drain(node *core_v1.Node) error {
	err := lc.LifeCycler.Drain(node)
	if err == nil {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeNormal,
			events.SuccessfulDrainNode,
			"Preparing upgrade for node: %v. Successfully drained node.",
			node.GetName())
	} else {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeNormal,
			events.FailedDrainNode,
			"Preparing upgrade for node: %v. Failed to drain node: %v",
			node.GetName(),
			err)
	}
	return err
}

// Reboot writes an Event
func (lc *EventingLifeCycler) Reboot(node *core_v1.Node) error {
	err := lc.LifeCycler.Reboot(node)
	if err == nil {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeNormal,
			events.SuccessfulRebootNode,
			"Upgrading OS for node: %v. Reboot successful.",
			node.GetName())
	} else {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeNormal,
			events.FailedRebootNode,
			"Upgrading OS for node: %v. Reboot failed: %v",
			node.GetName(),
			err)
	}
	return err
}

// Replace writes an Event
func (lc *EventingLifeCycler) Replace(node *core_v1.Node) error {
	err := lc.LifeCycler.Reboot(node)
	if err == nil {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeNormal,
			events.SuccessfulReplaceNode,
			"Replacing node for upgrade: %v. Termination successful",
			node.GetName())
	} else {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeNormal,
			events.FailedReplaceNode,
			"Replacing node for upgrade: %v. Termination failed: %v",
			node.GetName(),
			err)
	}
	return err
}

// Drain collects metrics
func (lc *InstrumentingLifeCycler) Drain(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		lc.Latency.With(
			prometheus.Labels{
				"controller": "servicing",
				"method":     "Drain",
			}).Observe(time.Since(begin).Seconds())

		lc.Total.With(
			prometheus.Labels{
				"controller": "servicing",
				"method":     "Drain",
			}).Add(1)

		if err != nil {
			lc.Failed.With(
				prometheus.Labels{
					"controller": "servicing",
					"method":     "Drain",
				}).Add(1)
		} else {
			lc.Successful.With(
				prometheus.Labels{
					"controller": "servicing",
					"method":     "Drain",
				}).Add(1)
		}
	}(time.Now())
	return lc.LifeCycler.Drain(node)
}

// Reboot collects metrics
func (lc *InstrumentingLifeCycler) Reboot(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		lc.Latency.With(
			prometheus.Labels{
				"controller": "servicing",
				"method":     "Reboot",
			}).Observe(time.Since(begin).Seconds())

		lc.Total.With(
			prometheus.Labels{
				"controller": "servicing",
				"method":     "Reboot",
			}).Add(1)

		if err != nil {
			lc.Failed.With(
				prometheus.Labels{
					"controller": "servicing",
					"method":     "Reboot",
				}).Add(1)
		} else {
			lc.Successful.With(
				prometheus.Labels{
					"controller": "servicing",
					"method":     "Reboot",
				}).Add(1)
		}
	}(time.Now())
	return lc.LifeCycler.Reboot(node)
}

// Replace collects metrics
func (lc *InstrumentingLifeCycler) Replace(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		lc.Latency.With(
			prometheus.Labels{
				"controller": "servicing",
				"method":     "Replace",
			}).Observe(time.Since(begin).Seconds())

		lc.Total.With(
			prometheus.Labels{
				"controller": "servicing",
				"method":     "Replace",
			}).Add(1)

		if err != nil {
			lc.Failed.With(
				prometheus.Labels{
					"controller": "servicing",
					"method":     "Replace",
				}).Add(1)
		} else {
			lc.Successful.With(
				prometheus.Labels{
					"controller": "servicing",
					"method":     "Replace",
				}).Add(1)
		}
	}(time.Now())
	return lc.LifeCycler.Replace(node)
}
