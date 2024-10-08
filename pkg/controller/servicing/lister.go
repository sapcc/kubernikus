package servicing

import (
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/coreos"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/flatcar"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/version"
)

const (
	AnnotationNodeForceReplace = "kubernikus.cloud.sap/forceReplace"
	AnnotationNodeSkipReplace  = "kubernikus.cloud.sap/skipReplace"
	LabelMaintenanceController = "cloud.sap/maintenance-profile"
)

type (
	// Lister enumerates Nodes in various states
	Lister interface {
		All() []*core_v1.Node
		Reboot() []*core_v1.Node
		Replace() []*core_v1.Node
		Updating() []*core_v1.Node
		Successful() []*core_v1.Node
		Failed() []*core_v1.Node
		NotReady() []*core_v1.Node
		Maintained() []*core_v1.Node
	}

	// ListerFactory produces a Lister
	ListerFactory interface {
		Make(*v1.Kluster) (Lister, error)
	}

	// NodeListerFactory produces a NodeLister
	NodeListerFactory struct {
		Logger            log.Logger
		NodeObservatory   *nodeobservatory.NodeObservatory
		CoreOSVersion     *coreos.Version
		CoreOSRelease     *coreos.Release
		FlatcarVersion    *flatcar.Version
		FlatcarRelease    *flatcar.Release
		NodeUpdateHoldoff time.Duration
	}

	// NodeLister knows how to figure out the state of Nodes
	NodeLister struct {
		Logger            log.Logger
		Kluster           *v1.Kluster
		Lister            listers_core_v1.NodeLister
		CoreOSVersion     *coreos.Version
		CoreOSRelease     *coreos.Release
		FlatcarVersion    *flatcar.Version
		FlatcarRelease    *flatcar.Release
		NodeUpdateHoldoff time.Duration
	}

	// LoggingLister writes log messages
	LoggingLister struct {
		Lister Lister
		Logger log.Logger
	}
)

// NewNodeListerFactory produces a new factory
func NewNodeListerFactory(logger log.Logger, recorder record.EventRecorder, factories config.Factories, clients config.Clients, holdoff time.Duration) ListerFactory {
	return &NodeListerFactory{
		Logger:            logger,
		NodeObservatory:   factories.NodesObservatory.NodeInformer(),
		CoreOSVersion:     &coreos.Version{},
		CoreOSRelease:     &coreos.Release{},
		FlatcarVersion:    &flatcar.Version{},
		FlatcarRelease:    &flatcar.Release{},
		NodeUpdateHoldoff: holdoff,
	}
}

// Make a NodeListerFactory
func (f *NodeListerFactory) Make(k *v1.Kluster) (Lister, error) {
	var lister Lister
	logger := log.With(f.Logger, "kluster", k.Spec.Name, "project", k.Account())

	klusterLister, err := f.NodeObservatory.GetListerForKluster(k)
	if err != nil {
		return lister, errors.Wrap(err, "Couldn't create NodeLister from NodeObservatory")
	}

	lister = &NodeLister{
		Logger:            logger,
		Kluster:           k,
		Lister:            klusterLister,
		CoreOSVersion:     f.CoreOSVersion,
		CoreOSRelease:     f.CoreOSRelease,
		FlatcarVersion:    f.FlatcarVersion,
		FlatcarRelease:    f.FlatcarRelease,
		NodeUpdateHoldoff: f.NodeUpdateHoldoff,
	}

	lister = &LoggingLister{
		Lister: lister,
		Logger: logger,
	}

	return lister, nil
}

// All nodes
func (d *NodeLister) All() []*core_v1.Node {
	nodes, err := d.Lister.List(labels.Everything())
	if err != nil {
		d.Logger.Log(
			"msg", "Couldn't list nodes. Skipping OS upgrade.",
			"err", err,
		)
		return []*core_v1.Node{}
	}
	return nodes
}

// Reboot lists nodes that have an outdated OS version
func (d *NodeLister) Reboot() []*core_v1.Node {
	var rebootable, found []*core_v1.Node

	latestFlatcar, err := d.FlatcarVersion.Stable()
	if err != nil {
		d.Logger.Log(
			"msg", "Couldn't get Flatcar version.",
			"err", err,
		)
		return found
	}

	releasedFlatcar, err := d.FlatcarRelease.GrownUp(latestFlatcar, d.NodeUpdateHoldoff)
	if err != nil {
		d.Logger.Log(
			"msg", "Couldn't get Flatcar releases.",
			"err", err,
		)
	}

	for _, pool := range d.Kluster.Spec.NodePools {
		for _, node := range d.All() {
			if util.IsKubernikusNode(node.Name, d.Kluster.Spec.Name, pool.Name) {
				if util.IsFlatcarNodeWithRkt(node) {
					continue
				}

				if *pool.Config.AllowReboot == true {
					rebootable = append(rebootable, node)
				}
			}
		}
	}

	for _, node := range rebootable {
		uptodate := true
		var err error

		if strings.HasPrefix(node.Status.NodeInfo.OSImage, "Flatcar Container Linux") {
			if releasedFlatcar {
				uptodate, err = d.FlatcarVersion.IsNodeUptodate(node)
			}
		} else {
			d.Logger.Log(
				"msg", "Unsupported OS on node. Skipping OS upgrade.",
				"os", node.Status.NodeInfo.OSImage,
			)
			continue
		}
		if err != nil {
			d.Logger.Log(
				"msg", "Couldn't get OS version from Node. Skipping OS upgrade.",
				"err", err,
			)
			continue
		}

		if !uptodate {
			found = append(found, node)
		}
	}

	return found
}

// Replacement lists nodes that have an outdated Kubelet/Kube-Proxy
func (d *NodeLister) Replace() []*core_v1.Node {
	var upgradable, found []*core_v1.Node
	var nodeNameToPool map[string]*models.NodePool
	nodeNameToPool = make(map[string]*models.NodePool)

	latestFlatcar, err := d.FlatcarVersion.Stable()
	if err != nil {
		d.Logger.Log(
			"msg", "Couldn't get Flatcar version.",
			"err", err,
		)
		return found
	}

	releasedFlatcar, err := d.FlatcarRelease.GrownUp(latestFlatcar, d.NodeUpdateHoldoff)
	if err != nil {
		d.Logger.Log(
			"msg", "Couldn't get Flatcar releases.",
			"err", err,
		)
	}

	for i, pool := range d.Kluster.Spec.NodePools {
		for _, node := range d.All() {
			if util.IsKubernikusNode(node.Name, d.Kluster.Spec.Name, pool.Name) {
				nodeNameToPool[node.GetName()] = &d.Kluster.Spec.NodePools[i]

				if *pool.Config.AllowReplace == true || util.IsFlatcarNodeWithRkt(node) || util.EnabledValue(node.Annotations[AnnotationNodeForceReplace]) {
					upgradable = append(upgradable, node)
				}
			}
		}
	}

	klusterVersion, err := version.ParseSemantic(d.Kluster.Status.ApiserverVersion)
	if err != nil {
		klusterVersion = nil
	}

	for _, node := range upgradable {
		if util.EnabledValue(node.Annotations[AnnotationNodeSkipReplace]) {
			continue
		}

		if util.EnabledValue(node.Annotations[AnnotationNodeForceReplace]) {
			found = append(found, node)
			continue
		}

		if util.IsCoreOSNode(node) && util.IsFlatcarNodePool(nodeNameToPool[node.GetName()]) {
			found = append(found, node)
			continue
		}

		if klusterVersion == nil {
			d.Logger.Log(
				"msg", "Couldn't parse Kluster version. Skipping node upgrades because of missing api version.",
				"node", node.GetName(),
				"err", err,
			)
			continue
		}

		kubeletVersion, err := getKubeletVersion(node)
		if err != nil {
			d.Logger.Log(
				"msg", "Couldn't get Kubelet version from Node. Skipping node upgrade.",
				"node", node.GetName(),
				"err", err,
			)
			continue
		}

		if kubeletVersion.LessThan(klusterVersion) {
			found = append(found, node)
			continue
		}

		if util.IsFlatcarNodeWithRkt(node) {
			uptodate := true

			if strings.HasPrefix(node.Status.NodeInfo.OSImage, "Flatcar Container Linux") {
				if releasedFlatcar {
					uptodate, err = d.FlatcarVersion.IsNodeUptodate(node)
				}
			} else {
				d.Logger.Log(
					"msg", "Unsupported OS on node. Skipping OS upgrade.",
					"os", node.Status.NodeInfo.OSImage,
				)
				continue
			}
			if err != nil {
				d.Logger.Log(
					"msg", "Couldn't get OS version from Node. Skipping OS upgrade.",
					"err", err,
				)
				continue
			}

			if !uptodate {
				found = append(found, node)
			}
		}
	}

	return found
}

// NotReady lists nodes which are not ready
func (d *NodeLister) NotReady() []*core_v1.Node {
	return d.withCondidtion(
		core_v1.NodeReady,
		core_v1.ConditionFalse,
		core_v1.ConditionUnknown)
}

// Updating lists nodes which are being updated
func (d *NodeLister) Updating() []*core_v1.Node {
	return d.hasAnnotation(AnnotationUpdateTimestamp)
}

// Successful lists nodes which have been successfully updated
func (d *NodeLister) Successful() []*core_v1.Node {
	var found []*core_v1.Node

	// Node must have updating annotation
	// Node must not be in the list of nodes to be rebooted
	// Node must not be in the list of nodes to be replaced
	// Node must be ready

	for _, node := range d.Updating() {
		failure := false

		for _, r := range d.Reboot() {
			if r == node {
				failure = true
				break
			}
		}

		if failure {
			continue
		}

		for _, r := range d.Replace() {
			if r == node {
				failure = true
				break
			}
		}

		if failure {
			continue
		}

		_, condition := getNodeCondition(&node.Status, core_v1.NodeReady)
		if condition.Status == core_v1.ConditionTrue {
			found = append(found, node)
		}
	}

	return found
}

// Failed lists nodes which failed to be updated
func (d *NodeLister) Failed() []*core_v1.Node {
	var found []*core_v1.Node

	// is beyond update timeout AND (
	//	 is to be rebooted OR
	//   is to be replaces OR
	//   is unhealthy
	// )

	for _, node := range d.updateTimeout() {
		failed := false
		for _, r := range d.Replace() {
			if r == node {
				failed = true
				found = append(found, node)
				break
			}
		}

		if failed {
			continue
		}

		for _, r := range d.Reboot() {
			if r == node {
				failed = true
				found = append(found, node)
				break
			}
		}

		if failed {
			continue
		}

		_, condition := getNodeCondition(&node.Status, core_v1.NodeReady)
		if condition.Status == core_v1.ConditionFalse {
			found = append(found, node)
		}
	}

	return found
}

// Returns all nodes that are assumed to be maintained by the maintenance-controller.
func (d *NodeLister) Maintained() []*core_v1.Node {
	var found []*core_v1.Node

	for _, node := range d.All() {
		_, ok := node.Labels[LabelMaintenanceController]
		if ok {
			found = append(found, node)
		}
	}

	return found
}

func (d *NodeLister) updateTimeout() []*core_v1.Node {
	var found []*core_v1.Node

	for _, node := range d.hasAnnotation(AnnotationUpdateTimestamp) {
		updateTime, ok := node.Annotations[AnnotationUpdateTimestamp]
		if !ok {
			continue
		}

		pt, err := time.Parse(time.RFC3339, updateTime)
		if err != nil {
			d.Logger.Log(
				"msg", "failed to parse updatetime annotation",
				"node", node.GetName(),
				"err", err,
			)
			continue
		}

		timeout := pt.Add(UpdateTimeout)

		if Now().After(timeout) {
			found = append(found, node)
		}
	}

	return found
}

func (d *NodeLister) withCondidtion(conditionType core_v1.NodeConditionType,
	expected ...core_v1.ConditionStatus) []*core_v1.Node {
	var found []*core_v1.Node

	for _, node := range d.All() {
		_, condition := getNodeCondition(&node.Status, conditionType)
		if condition == nil || condition.Type != conditionType {
			continue
		}

		for _, e := range expected {
			if condition.Status == e {
				found = append(found, node)
				break
			}
		}
	}

	return found
}

func (d *NodeLister) hasAnnotation(name string) []*core_v1.Node {
	var found []*core_v1.Node

	for _, node := range d.All() {
		_, ok := node.ObjectMeta.Annotations[name]
		if !ok {
			continue
		}

		found = append(found, node)
	}

	return found
}

// All logs
func (l *LoggingLister) All() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing all nodes",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.All()
}

// Reboot logs
func (l *LoggingLister) Reboot() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing nodes requiring reboot",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.Reboot()
}

// Replacement logs
func (l *LoggingLister) Replace() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing nodes requiring replacement",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.Replace()
}

// NotReady logs
func (l *LoggingLister) NotReady() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing not ready nodes",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.NotReady()
}

// Updating logs
func (l *LoggingLister) Updating() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing updating nodes",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.Updating()
}

// Successful logs
func (l *LoggingLister) Successful() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing successfully updated nodes",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.Successful()
}

// Failed logs
func (l *LoggingLister) Failed() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing unsuccessfully updated nodes",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.Failed()
}

func (l *LoggingLister) Maintained() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing nodes assumed to be maintained by the maintenance-controller",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.Maintained()
}

func getKubeletVersion(node *core_v1.Node) (*version.Version, error) {
	return version.ParseSemantic(node.Status.NodeInfo.KubeletVersion)
}

// GetNodeCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func getNodeCondition(status *core_v1.NodeStatus,
	conditionType core_v1.NodeConditionType) (int, *core_v1.NodeCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}
