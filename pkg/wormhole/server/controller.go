package server

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/databus23/guttle"
	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/util/iptables"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	utilexec "k8s.io/utils/exec"
)

const (
	KUBERNIKUS_TUNNELS iptables.Chain = "KUBERNIKUS-TUNNELS"
)

type Controller struct {
	nodes       informers.NodeInformer
	tunnel      *guttle.Server
	queue       workqueue.RateLimitingInterface
	store       map[string][]route
	iptables    iptables.Interface
	hijackPort  int
	serviceCIDR string
}

type route struct {
	cidr       string
	identifier string
}

func NewController(informer informers.NodeInformer, serviceCIDR string, tunnel *guttle.Server) *Controller {
	c := &Controller{
		nodes:       informer,
		tunnel:      tunnel,
		queue:       workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		store:       make(map[string][]route),
		iptables:    iptables.New(utilexec.New(), iptables.ProtocolIpv4),
		hijackPort:  9191,
		serviceCIDR: serviceCIDR,
	}

	c.nodes.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.queue.Add(key)
			}
		},
	})

	return c
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer c.queue.ShutDown()
	defer wg.Done()
	wg.Add(1)
	glog.Infof(`Starting WormholeGenerator with %d workers`, threadiness)

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				glog.V(5).Infof("Running periodic recheck. Queuing all known nodes...")
				for key, _ := range c.store {
					c.queue.Add(key)
				}
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	<-stopCh
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.reconcile(key.(string))
	c.handleErr(err, key)
	return true
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}
	glog.Errorf("Requeuing %v: %v", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	glog.Infof("Dropping %v. Too many errors", key)
	c.queue.Forget(key)
}

func (c *Controller) reconcile(key string) error {
	obj, exists, err := c.nodes.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		return c.delNode(key)
	}

	return c.addNode(key, obj.(*v1.Node))
}

func (c *Controller) addNode(key string, node *v1.Node) error {

	identifier := fmt.Sprintf("system:node:%v", node.GetName())
	glog.Infof("Adding tunnel routes for node %v", identifier)

	podCIDR := node.Spec.PodCIDR

	ip, err := GetNodeHostIP(node)
	if err != nil {
		return err
	}
	nodeCIDR := ip.String() + "/32"

	if err := c.tunnel.AddClientRoute(podCIDR, identifier); err != nil {
		return err
	}
	c.store[key] = append(c.store[key], route{cidr: podCIDR, identifier: identifier})
	if err := c.tunnel.AddRoute(podCIDR); err != nil {
		return err
	}
	if err := c.tunnel.AddClientRoute(nodeCIDR, identifier); err != nil {
		return err
	}
	c.store[key] = append(c.store[key], route{cidr: nodeCIDR, identifier: identifier})
	if err := c.tunnel.AddRoute(nodeCIDR); err != nil {
		return err
	}

	return c.redoIPTablesSpratz()
}

func (c *Controller) delNode(key string) error {
	routes := c.store[key]
	for _, route := range routes {
		c.tunnel.DeleteClientRoute(route.cidr, route.identifier)
		c.tunnel.DeleteRoute(route.cidr)
	}
	return c.redoIPTablesSpratz()
}

func (c *Controller) redoIPTablesSpratz() error {
	table := iptables.TableNAT

	if _, err := c.iptables.EnsureChain(table, KUBERNIKUS_TUNNELS); err != nil {
		glog.Errorf("Failed to ensure that %s chain %s exists: %v", table, KUBERNIKUS_TUNNELS, err)
		return err
	}

	args := []string{"-m", "comment", "--comment", "kubernikus tunnels", "-j", string(KUBERNIKUS_TUNNELS)}
	if _, err := c.iptables.EnsureRule(iptables.Append, table, iptables.ChainOutput, args...); err != nil {
		glog.Errorf("Failed to ensure that %s chain %s jumps to %s: %v", table, iptables.ChainOutput, KUBERNIKUS_TUNNELS, err)
		return err
	}

	iptablesSaveRaw := bytes.NewBuffer(nil)
	existingNatChains := make(map[iptables.Chain]string)
	err := c.iptables.SaveInto(table, iptablesSaveRaw)
	if err != nil {
		glog.Errorf("Failed to execute iptables-save, syncing all rules: %v", err)
	} else {
		existingNatChains = iptables.GetChainLines(table, iptablesSaveRaw.Bytes())
	}

	natChains := bytes.NewBuffer(nil)
	natRules := bytes.NewBuffer(nil)
	writeLine(natChains, "*nat")
	if chain, ok := existingNatChains[KUBERNIKUS_TUNNELS]; ok {
		writeLine(natChains, chain)
	} else {
		writeLine(natChains, iptables.MakeChainLine(KUBERNIKUS_TUNNELS))
	}

	for key, _ := range c.store {
		err := c.writeTunnelRedirect(key, natRules)
		if err != nil {
			return err
		}
	}

	writeLine(natRules,
		"-A", string(KUBERNIKUS_TUNNELS),
		"-m", "comment", "--comment", `"cluster service CIDR tunnel redirect"`,
		"--dst", c.serviceCIDR,
		"-p", "tcp",
		"-j", "REDIRECT",
		"--to-ports", strconv.Itoa(c.hijackPort),
	)

	writeLine(natRules, "COMMIT")

	lines := append(natChains.Bytes(), natRules.Bytes()...)
	glog.V(6).Infof("Restoring iptables rules: %s", lines)
	err = c.iptables.RestoreAll(lines, iptables.NoFlushTables, iptables.RestoreCounters)
	if err != nil {
		glog.Errorf("Failed to execute iptables-restore: %v", err)
		return err
	}

	return nil
}

func (c *Controller) writeTunnelRedirect(key string, filterRules *bytes.Buffer) error {
	obj, exists, err := c.nodes.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	node := obj.(*v1.Node)
	ip, err := GetNodeHostIP(node)
	if err != nil {
		return err
	}

	port := strconv.Itoa(c.hijackPort)

	writeLine(filterRules,
		"-A", string(KUBERNIKUS_TUNNELS),
		"-m", "comment", "--comment", fmt.Sprintf(`"node ip tunnel redirect for %s"`, key),
		"--dst", ip.String(),
		"-p", "tcp",
		"-j", "REDIRECT",
		"--to-ports", port,
	)

	writeLine(filterRules,
		"-A", string(KUBERNIKUS_TUNNELS),
		"-m", "comment", "--comment", fmt.Sprintf(`"pod cidr tunnel redirect for %s"`, key),
		"--dst", node.Spec.PodCIDR,
		"-p", "tcp",
		"-j", "REDIRECT",
		"--to-ports", port,
	)

	return nil
}

func writeLine(buf *bytes.Buffer, words ...string) {
	buf.WriteString(strings.Join(words, " ") + "\n")
}

func GetNodeHostIP(node *v1.Node) (net.IP, error) {
	addresses := node.Status.Addresses
	addressMap := make(map[v1.NodeAddressType][]v1.NodeAddress)
	for i := range addresses {
		addressMap[addresses[i].Type] = append(addressMap[addresses[i].Type], addresses[i])
	}
	if addresses, ok := addressMap[v1.NodeInternalIP]; ok {
		return net.ParseIP(addresses[0].Address), nil
	}
	if addresses, ok := addressMap[v1.NodeExternalIP]; ok {
		return net.ParseIP(addresses[0].Address), nil
	}
	return nil, fmt.Errorf("host IP unknown; known addresses: %v", addresses)
}
