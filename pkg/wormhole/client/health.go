package client

import (
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"golang.org/x/net/ipv4"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	typed_core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/pkg/util/icmp"
	"github.com/sapcc/kubernikus/pkg/util/netutil"
)

var sequenceNumber uint32 = 0

const checkInterval = 60 * time.Second
const intervalJitter = 0.5

const NodeRouteBroken v1.NodeConditionType = "RouteBroken"

type healthChecker struct {
	nodeName string
	localIP  net.IP
	listener *icmp.Listener
	logger   kitlog.Logger
	client   typed_core_v1.NodeInterface
}

func NewHealthChecker(kubeconfig, context, nodeNameOverride string, logger kitlog.Logger) (*healthChecker, error) {
	var hc = healthChecker{logger: logger, nodeName: nodeNameOverride}

	client, err := kubernetes.NewClient(kubeconfig, context, logger)
	if err != nil {
		return nil, fmt.Errorf("Failed to create kubernetes client: %s", err)
	}
	hc.client = client.CoreV1().Nodes()

	hc.nodeName = nodeNameOverride

	interfaceName, err := netutil.DefaultInterfaceName()
	if err != nil {
		return nil, err
	}

	if hc.localIP, err = netutil.InterfaceAddress(interfaceName); err != nil {
		return nil, err
	}

	return &hc, nil
}

func (hc *healthChecker) Start(stopCh <-chan struct{}) error {
	hc.logger.Log("msg", "Starting healthecker", "interval", checkInterval, "jitter", intervalJitter)

	var err error
	if hc.listener, err = icmp.NewListener(hc.localIP.String()); err != nil {
		return err
	}

	wait.JitterUntil(hc.Reconcile, checkInterval, intervalJitter, true, stopCh)

	return nil
}

func (hc *healthChecker) Reconcile() {
	if err := hc.reconcile(); err != nil {
		hc.logger.Log("msg", "healthcheck", "err", err)
	}
}

func (hc *healthChecker) reconcile() error {

	destinationIP, err := netutil.InterfaceAddress("cbr0")
	if err != nil {
		return fmt.Errorf("Interface cbr0 not found: %s", err)
	}

	nodeName, err := hc.myNodeName()
	if err != nil {
		return fmt.Errorf("Failed to discover own node name: %s", err)
	}

	node, err := hc.client.Get(nodeName, meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Couldn't get node %s from api: %s", nodeName, err)
	}

	newCondition := v1.NodeCondition{Type: NodeRouteBroken}

	backoff := wait.Backoff{Duration: 1 * time.Second, Jitter: 0.2, Steps: 1, Factor: 2}
	//If the Node was previously working we retry a few times
	if !IsNodeRouteBroken(node) {
		backoff.Steps = 3
	}

	err = wait.ExponentialBackoff(backoff, func() (bool, error) {
		return checkRedirect(hc.listener, hc.localIP, destinationIP), nil
	})

	if err == nil {
		newCondition.Status = v1.ConditionFalse
		newCondition.Reason = "RedirectOK"
		if IsNodeRouteBroken(node) {
			newCondition.LastTransitionTime = meta_v1.NewTime(time.Now())
		}
	} else {
		newCondition.Status = v1.ConditionTrue
		newCondition.Reason = "RedirectFailed"
		if !IsNodeRouteBroken(node) {
			newCondition.LastTransitionTime = meta_v1.NewTime(time.Now())
		}
	}

	if err := hc.SetConditions([]v1.NodeCondition{newCondition}, nodeName); err != nil {
		return fmt.Errorf("Failed to update condition for node %s: %s", nodeName, err)
	}
	hc.logger.Log("node", nodeName, "check", newCondition.Type, "status", newCondition.Status, "v", 2)

	return nil
}

func (hc *healthChecker) SetConditions(newConditions []v1.NodeCondition, node string) error {
	for i := range newConditions {
		// Each time we update the conditions, we update the heart beat time
		newConditions[i].LastHeartbeatTime = meta_v1.NewTime(time.Now())
	}
	patch, err := generatePatch(newConditions)
	if err != nil {
		return err
	}
	_, err = hc.client.PatchStatus(node, patch)
	return err
}

func (hc *healthChecker) myNodeName() (string, error) {
	if hc.nodeName != "" {
		return hc.nodeName, nil
	}
	list, err := hc.client.List(meta_v1.ListOptions{})
	if err != nil {
		return "", err
	}
	me, err := util.ThisNode(list.Items)
	if err != nil {
		return "", err
	}
	hc.nodeName = me.Name
	return hc.nodeName, nil
}

func checkRedirect(listener *icmp.Listener, expectedNextHop, dest net.IP) bool {

	//get a new sequence number
	seq := atomic.AddUint32(&sequenceNumber, 1)
	listener.SendEcho(&net.IPAddr{IP: dest}, int(seq))
	listener.SetReadDeadline(time.Now().Add(1 * time.Second))

	for {
		msg, err := listener.Read()
		if err != nil {
			return false
		}
		if msg.Type == ipv4.ICMPTypeRedirect && msg.Body.(*icmp.Redirect).NextHop.Equal(expectedNextHop) {
			return true
		}
	}
}

// generatePatch generates condition patch
func generatePatch(conditions []v1.NodeCondition) ([]byte, error) {
	raw, err := json.Marshal(&conditions)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf(`{"status":{"conditions":%s}}`, raw)), nil
}

func IsNodeRouteBroken(node *v1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == NodeRouteBroken {
			return c.Status == v1.ConditionTrue
		}
	}
	return false
}
