package servicing

import (
	"fmt"
	"regexp"
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
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/events"
	"github.com/sapcc/kubernikus/pkg/controller/metrics"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/drain"
	"github.com/sapcc/kubernikus/pkg/util"
)

const (
	// EvictionTimeout defines when to abort the draining of a node
	EvictionTimeout = 4 * time.Minute
)

type (
	// LifeCycler managed a node's lifecycle actions
	LifeCycler interface {
		Drain(node *core_v1.Node) error
		Uncordon(node *core_v1.Node) error
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

// NewNodeLifeCyclerFactory produces a new factory
func NewNodeLifeCyclerFactory(logger log.Logger, recorder record.EventRecorder, factories config.Factories, clients config.Clients) LifeCyclerFactory {
	return &NodeLifeCyclerFactory{
		Logger:     logger,
		Recorder:   recorder,
		Satellites: clients.Satellites,
		Openstack:  factories.Openstack,
	}
}

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
	if err := lc.setUpdatingAnnotation(node); err != nil {
		return errors.Wrap(err, "Failed to drain node")
	}

	options := &drain.DrainOptions{
		Force:              true,
		IgnoreDaemonsets:   true,
		GracePeriodSeconds: -1,
		Timeout:            EvictionTimeout,
		DeleteLocalData:    true,
		Namespace:          meta_v1.NamespaceAll,
		Selector:           nil,
		Logger:             log.With(lc.Logger, "node", node.GetName()),
	}
	if err := drain.Drain(lc.Kubernetes, []*core_v1.Node{node}, options); err != nil {
		return errors.Wrap(err, "Failed to drain node")
	}
	return nil
}

// Reboot a node softly
func (lc *NodeLifeCycler) Reboot(node *core_v1.Node) error {
	id, err := instanceIDFromProviderID(node.Spec.ProviderID)
	if err != nil {
		return errors.Wrap(err, "rebooting node failed")
	}

	if err := lc.Openstack.RebootNode(id); err != nil {
		return errors.Wrap(err, "rebooting node failed")
	}

	return nil
}

// Replace a node by temrinating it
func (lc *NodeLifeCycler) Replace(node *core_v1.Node) error {
	id, err := instanceIDFromProviderID(node.Spec.ProviderID)
	if err != nil {
		return errors.Wrap(err, "deleting node failed")
	}

	if err := lc.Openstack.DeleteNode(id); err != nil {
		return errors.Wrap(err, "deleting node failed")
	}
	return nil
}

// Uncordon removes the updating annotation and uncordons the node
func (lc *NodeLifeCycler) Uncordon(node *core_v1.Node) error {
	if err := lc.removeUpdatingAnnotation(node); err != nil {
		return errors.Wrap(err, "failed to uncordon node")
	}
	if err := drain.Uncordon(lc.Kubernetes.Core().Nodes(), node, lc.Logger); err != nil {
		return errors.Wrap(err, "failed to uncordon node")
	}
	return nil
}

func (lc *NodeLifeCycler) setUpdatingAnnotation(node *core_v1.Node) error {
	if err := util.AddNodeAnnotation(node.Name, AnnotationUpdateTimestamp, Now().UTC().Format(time.RFC3339), lc.Kubernetes); err != nil {
		return errors.Wrap(err, "failed to set updating annotation")
	}
	return nil
}

func (lc *NodeLifeCycler) removeUpdatingAnnotation(node *core_v1.Node) error {
	if err := util.RemoveNodeAnnotation(node.Name, AnnotationUpdateTimestamp, lc.Kubernetes); err != nil {
		return errors.Wrap(err, "failed to remove updating annotation")
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

// Replace logs the action
func (lc *LoggingLifeCycler) Uncordon(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		lc.Logger.Log(
			"msg", "uncordoning node",
			"node", node.GetName(),
			"took", time.Since(begin),
			"v", 1,
			"err", err,
		)
	}(time.Now())
	return lc.LifeCycler.Uncordon(node)
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
			core_v1.EventTypeWarning,
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
			core_v1.EventTypeWarning,
			events.FailedRebootNode,
			"Upgrading OS for node: %v. Reboot failed: %v",
			node.GetName(),
			err)
	}
	return err
}

// Replace writes an Event
func (lc *EventingLifeCycler) Replace(node *core_v1.Node) error {
	err := lc.LifeCycler.Replace(node)
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
			core_v1.EventTypeWarning,
			events.FailedReplaceNode,
			"Replacing node for upgrade: %v. Termination failed: %v",
			node.GetName(),
			err)
	}
	return err
}

// Uncordon writes an Event
func (lc *EventingLifeCycler) Uncordon(node *core_v1.Node) error {
	err := lc.LifeCycler.Uncordon(node)
	if err == nil {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeNormal,
			events.SuccessfulRebootNode,
			"Uncordoning node: %v. Update was successful",
			node.GetName())
	} else {
		lc.Recorder.Eventf(
			lc.Kluster,
			core_v1.EventTypeWarning,
			events.FailedRebootNode,
			"Uncordoning node failed: %v. Update was successful anyway",
			node.GetName(),
			err)
	}
	return err
}

// Drain collects metrics
func (lc *InstrumentingLifeCycler) Drain(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		labels := prometheus.Labels{
			"controller": "servicing",
			"method":     "Drain",
		}

		lc.Latency.With(labels).Observe(time.Since(begin).Seconds())
		lc.Total.With(labels).Add(1)

		if err != nil {
			lc.Failed.With(labels).Add(1)
		} else {
			lc.Successful.With(labels).Add(1)
		}
	}(time.Now())
	return lc.LifeCycler.Drain(node)
}

// Reboot collects metrics
func (lc *InstrumentingLifeCycler) Reboot(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		labels := prometheus.Labels{
			"controller": "servicing",
			"method":     "Reboot",
		}

		lc.Latency.With(labels).Observe(time.Since(begin).Seconds())
		lc.Total.With(labels).Add(1)

		if err != nil {
			lc.Failed.With(labels).Add(1)
		} else {
			lc.Successful.With(labels).Add(1)
		}
	}(time.Now())
	return lc.LifeCycler.Reboot(node)
}

// Replace collects metrics
func (lc *InstrumentingLifeCycler) Replace(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		labels := prometheus.Labels{
			"controller": "servicing",
			"method":     "Drain",
		}

		lc.Latency.With(labels).Observe(time.Since(begin).Seconds())
		lc.Total.With(labels).Add(1)

		if err != nil {
			lc.Failed.With(labels).Add(1)
		} else {
			lc.Successful.With(labels).Add(1)
		}
	}(time.Now())
	return lc.LifeCycler.Replace(node)
}

// Uncordon collects metrics
func (lc *InstrumentingLifeCycler) Uncordon(node *core_v1.Node) (err error) {
	defer func(begin time.Time) {
		labels := prometheus.Labels{
			"controller": "servicing",
			"method":     "Uncordon",
		}

		lc.Latency.With(labels).Observe(time.Since(begin).Seconds())
		lc.Total.With(labels).Add(1)

		if err != nil {
			lc.Failed.With(labels).Add(1)
		} else {
			lc.Successful.With(labels).Add(1)
		}
	}(time.Now())
	return lc.LifeCycler.Uncordon(node)
}

// instanceIDFromProviderID splits a provider's id and return instanceID.
// A providerID is build out of '${ProviderName}:///${instance-id}'which contains ':///'.
// See cloudprovider.GetInstanceProviderID and Instances.InstanceID.
func instanceIDFromProviderID(providerID string) (instanceID string, err error) {
	var providerIDRegexp = regexp.MustCompile(`^openstack:///([^/]+)$`)

	matches := providerIDRegexp.FindStringSubmatch(providerID)
	if len(matches) != 2 {
		return "", fmt.Errorf("ProviderID \"%s\" didn't match expected format \"openstack:///InstanceID\"", providerID)
	}
	return matches[1], nil
}
