package server

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/koding/tunnel"
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
	nodes    informers.NodeInformer
	tunnel   *tunnel.Server
	queue    workqueue.RateLimitingInterface
	store    map[string]net.Listener
	iptables iptables.Interface
}

func NewController(informer informers.NodeInformer, tunnel *tunnel.Server) *Controller {
	c := &Controller{
		nodes:    informer,
		tunnel:   tunnel,
		queue:    workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		store:    make(map[string]net.Listener),
		iptables: iptables.New(utilexec.New(), iptables.ProtocolIpv4),
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
	if c.store[key] == nil {

		listener, err := net.Listen("tcp", "127.0.0.1:")
		if err != nil {
			return err
		}

		identifier := fmt.Sprintf("system:node:%v", node.GetName())
		glog.Infof("Listening to node %v on %v", identifier, listener.Addr())

		c.store[key] = listener
		c.tunnel.AddAddr(listener, nil, identifier)
		c.tunnel.AddHost(identifier, identifier)

		if err := c.redoIPTablesSpratz(); err != nil {
			return err
		}
	} else {
		glog.V(5).Infof("Already listening on this node... Skipping %v", key)
	}
	return nil
}

func (c *Controller) delNode(key string) error {
	listener := c.store[key]
	if listener != nil {
		glog.Infof("Deleting node %v", key)
		c.tunnel.DeleteAddr(listener, nil)
		listener.Close()
		c.store[key] = nil

		if err := c.redoIPTablesSpratz(); err != nil {
			return err
		}
	} else {
		glog.V(5).Infof("Not listening on this node... Skipping %v", key)
	}
	return nil
}

func (c *Controller) redoIPTablesSpratz() error {
	table := iptables.TableNAT

	if _, err := c.iptables.EnsureChain(table, KUBERNIKUS_TUNNELS); err != nil {
		glog.Errorf("Failed to ensure that %s chain %s exists: %v", table, KUBERNIKUS_TUNNELS, err)
		return err
	}

	args := []string{"-m", "comment", "--comment", "kubernikus tunnels", "-j", string(KUBERNIKUS_TUNNELS)}
	if _, err := c.iptables.EnsureRule(iptables.Append, table, iptables.ChainPrerouting, args...); err != nil {
		glog.Errorf("Failed to ensure that %s chain %s jumps to %s: %v", table, iptables.ChainPrerouting, KUBERNIKUS_TUNNELS, err)
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

	port := c.store[key].Addr().(*net.TCPAddr).Port

	writeLine(filterRules,
		"-A", string(KUBERNIKUS_TUNNELS),
		"-m", "comment", "--comment", key,
		"--dst", ip.String(),
		"-p", "tcp",
		"--dport", "22",
		"-j", "REDIRECT",
		"--to-ports", fmt.Sprintf("%v", port),
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

