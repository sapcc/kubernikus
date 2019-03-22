package servicing

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"

	"github.com/go-kit/kit/log"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/util/generator"
	"github.com/sapcc/kubernikus/pkg/util/version"
)

type (
	// Lister enumerates Nodes in various states
	Lister interface {
		All() []*core_v1.Node
		RequiringReboot() []*core_v1.Node
		RequiringReplacement() []*core_v1.Node
		NotReady() []*core_v1.Node
	}

	// ListerFactory produces a Lister
	ListerFactory interface {
		Make(*v1.Kluster) (Lister, error)
	}

	// NodeListerFactory produces a NodeLister
	NodeListerFactory struct {
		Logger          log.Logger
		NodeObservatory *nodeobservatory.NodeObservatory
		CoreOSVersion   *LatestCoreOSVersion
	}

	// NodeLister knows how to figure out the state of Nodes
	NodeLister struct {
		Logger        log.Logger
		Kluster       *v1.Kluster
		Lister        listers_core_v1.NodeLister
		CoreOSVersion *LatestCoreOSVersion
	}

	// LoggingLister writes log messages
	LoggingLister struct {
		Lister Lister
		Logger log.Logger
	}
)

// Make a NodeListerFactory
func (f *NodeListerFactory) Make(k *v1.Kluster) (Lister, error) {
	var lister Lister
	logger := log.With(f.Logger, "kluster", k.Spec.Name, "project", k.Account())

	klusterLister, err := f.NodeObservatory.GetListerForKluster(k)
	if err != nil {
		return lister, errors.Wrap(err, "Couldn't create NodeLister from NodeObservatory")
	}

	lister = &NodeLister{
		Logger:        logger,
		Kluster:       k,
		Lister:        klusterLister,
		CoreOSVersion: f.CoreOSVersion,
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

// RequiringReboot lists nodes that have an outdated CoreOS version
func (d *NodeLister) RequiringReboot() []*core_v1.Node {
	var rebootable, found []*core_v1.Node

	for _, pool := range d.Kluster.Spec.NodePools {
		if !pool.Config.AllowReboot {
			continue
		}

		prefix := fmt.Sprintf("%v-%v-", d.Kluster.Spec.Name, pool.Name)
		for _, node := range d.All() {
			if !strings.HasPrefix(node.GetName(), prefix) {
				continue
			}
			if len(node.GetName()) == len(prefix)+generator.RandomLength {
				rebootable = append(rebootable, node)
			}
		}
	}

	for _, node := range rebootable {
		uptodate, err := d.CoreOSVersion.IsNodeUptodate(node)
		if err != nil {
			d.Logger.Log(
				"msg", "Couldn't get CoreOS version from Node. Skipping OS upgrade.",
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

// RequiringReplacement lists nodes that have an outdated Kubelet/Kube-Proxy
func (d *NodeLister) RequiringReplacement() []*core_v1.Node {
	var upgradable, found []*core_v1.Node

	for _, pool := range d.Kluster.Spec.NodePools {
		if !pool.Config.AllowReplace {
			continue
		}

		prefix := fmt.Sprintf("%v-%v-", d.Kluster.Spec.Name, pool.Name)
		for _, node := range d.All() {
			if !strings.HasPrefix(node.GetName(), prefix) {
				continue
			}

			if len(node.GetName()) == len(prefix)+generator.RandomLength {
				upgradable = append(upgradable, node)
			}

		}
	}

	klusterVersion, err := version.ParseSemantic(d.Kluster.Status.Version)
	if err != nil {
		d.Logger.Log(
			"msg", "Couldn't parse Kluster version. Skipping node upgrade.",
			"err", err,
		)
		return found
	}

	for _, node := range upgradable {
		kubeletVersion, err := getKubeletVersion(node)
		if err != nil {
			d.Logger.Log(
				"msg", "Couldn't get Kubelet version from Node. Skipping node upgrade.",
				"err", err,
			)
			continue
		}

		if kubeletVersion.LessThan(klusterVersion) {
			found = append(found, node)
			continue
		}

		kubeProxyVersion, err := getKubeProxyVersion(node)
		if err != nil {
			d.Logger.Log(
				"msg", "Couldn't get KubeProxy version from Node. Skipping node upgrade.",
				"err", err,
			)
			continue
		}

		if kubeProxyVersion.LessThan(klusterVersion) {
			found = append(found, node)
			continue
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

// RequiringReboot logs
func (l *LoggingLister) RequiringReboot() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing nodes requiring reboot",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.RequiringReboot()
}

// RequiringReplacement logs
func (l *LoggingLister) RequiringReplacement() (nodes []*core_v1.Node) {
	defer func(begin time.Time) {
		l.Logger.Log(
			"msg", "listing nodes requiring replacement",
			"took", time.Since(begin),
			"count", len(nodes),
			"v", 3,
		)
	}(time.Now())
	return l.Lister.RequiringReplacement()
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

func getKubeletVersion(node *core_v1.Node) (*version.Version, error) {
	return version.ParseSemantic(node.Status.NodeInfo.KubeletVersion)
}

func getKubeProxyVersion(node *core_v1.Node) (*version.Version, error) {
	return version.ParseSemantic(node.Status.NodeInfo.KubeProxyVersion)
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
