package client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/databus23/guttle"
	"github.com/go-kit/kit/log"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	utilexec "k8s.io/utils/exec"

	"github.com/sapcc/kubernikus/pkg/util/iptables"
	"github.com/sapcc/kubernikus/pkg/wormhole"
)

const (
	KUBERNIKUS_TUNNELS iptables.Chain = "KUBERNIKUS-TUNNELS"
)

type Controller struct {
	nodes       informers.NodeInformer
	queue       workqueue.RateLimitingInterface
	store       map[string]route
	iptables    iptables.Interface
	serviceCIDR string
	Logger      log.Logger
	ClientCA    string
	Certificate string
	PrivateKey  string
}

type route struct {
	port       int
	client     *guttle.Client
	identifier string
}

func NewController(informer informers.NodeInformer, serviceCIDR string, logger log.Logger, clientCA string, certificate string, privateKey string) *Controller {
	logger = log.With(logger, "controller", "tunnel")
	c := &Controller{
		nodes:       informer,
		queue:       workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)),
		store:       make(map[string]route),
		iptables:    iptables.New(utilexec.New(), iptables.ProtocolIpv4, logger),
		serviceCIDR: serviceCIDR,
		Logger:      logger,
		ClientCA:    clientCA,
		Certificate: certificate,
		PrivateKey:  privateKey,
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
	c.Logger.Log(
		"msg", "starting WormholeGenerator",
		"threadiness", threadiness)

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.Logger.Log(
					"msg", "Running periodic recheck. Queuing all known nodes...",
					"v", 5)
				for key := range c.store {
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
	c.Logger.Log(
		"msg", "requeuing because of error",
		"key", key,
		"err", err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.Logger.Log(
		"msg", "dropping because of too many error",
		"key", key)
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
	_, found := c.store[key]
	if !found {
		identifier := fmt.Sprintf("system:node:%v", node.GetName())
		c.Logger.Log(
			"msg", "adding tunnel routes",
			"node", identifier)

		ip, err := GetNodeHostIP(node)
		if err != nil {
			return err
		}

		port, err := GetFreePort()
		if err != nil {
			return err
		}

		nodeClientOps := guttle.ClientOptions{
			ServerAddr: ip.String() + ":9090",
			ListenAddr: fmt.Sprintf("0.0.0.0:%d", port),
		}

		tlsConfig, err := wormhole.NewTLSConfig(c.Certificate, c.PrivateKey)
		if err != nil {
			c.Logger.Log(
				"msg", "Failed to load cert or key",
				"err", err,
			)
			return err
		}
		caPool, err := wormhole.LoadCAFile(c.ClientCA)
		if err != nil {
			c.Logger.Log(
				"msg", "Failed to load ca file",
				"file", c.ClientCA,
				"err", err,
			)
			return err
		}
		tlsConfig.RootCAs = caPool
		dialFunc := func(network, address string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			conn, err := tls.DialWithDialer(dialer, network, address, tlsConfig)
			if err != nil {
				c.Logger.Log(
					"msg", "failed to open connection",
					"address", address,
					"err", err)
			}
			return conn, err
		}
		nodeClientOps.Dial = dialFunc
		c.Logger.Log("msg", "Configured with tls dialer")

		nodeClient := guttle.NewClient(&nodeClientOps)

		//TODO use wg to properly clean goroutine at stop
		go nodeClient.Start()

		c.store[key] = route{
			client:     nodeClient,
			port:       port,
			identifier: identifier,
		}
	}

	return c.redoIPTablesSpratz()
}

func (c *Controller) delNode(key string) error {
	route := c.store[key]
	route.client.Stop()
	return c.redoIPTablesSpratz()
}

func (c *Controller) redoIPTablesSpratz() error {
	table := iptables.TableNAT

	if _, err := c.iptables.EnsureChain(table, KUBERNIKUS_TUNNELS); err != nil {
		c.Logger.Log(
			"msg", "failed to ensure that chain exists",
			"table", table,
			"chain", KUBERNIKUS_TUNNELS,
			"err", err)
		return err
	}

	args := []string{"-m", "comment", "--comment", "kubernikus tunnels", "-j", string(KUBERNIKUS_TUNNELS)}
	if _, err := c.iptables.EnsureRule(iptables.Append, table, iptables.ChainOutput, args...); err != nil {
		c.Logger.Log(
			"msg", "failed to ensure jump",
			"table", table,
			"target", iptables.ChainOutput,
			"chain", KUBERNIKUS_TUNNELS,
			"err", err)
		return err
	}

	iptablesSaveRaw := bytes.NewBuffer(nil)
	existingNatChains := make(map[iptables.Chain]string)
	err := c.iptables.SaveInto(table, iptablesSaveRaw)
	if err != nil {
		c.Logger.Log(
			"msg", "failed to execute iptables-save, syncing all rules",
			"err", err)
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

	c.Logger.Log(
		"msg", "iterating on store")
	for key, route := range c.store {
		err := c.writeTunnelRedirect(key, route.port, natRules)
		if err != nil {
			return err
		}
	}

	writeLine(natRules, "COMMIT")

	lines := append(natChains.Bytes(), natRules.Bytes()...)
	c.Logger.Log(
		"msg", "Restoring iptables rules",
		"rules", lines,
		"v", 6)

	err = c.iptables.RestoreAll(lines, iptables.NoFlushTables, iptables.RestoreCounters)
	if err != nil {
		c.Logger.Log(
			"msg", "Failed to execute iptables-restore",
			"err", err)
		return err
	}

	return nil
}

func (c *Controller) writeTunnelRedirect(key string, clientPort int, natRules *bytes.Buffer) error {
	c.Logger.Log(
		"msg", "writing tunnel redirect",
		"key", key)
	obj, exists, err := c.nodes.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		c.Logger.Log(
			"msg", "node does not exist")
		return nil
	}

	node := obj.(*v1.Node)
	ip, err := GetNodeHostIP(node)
	if err != nil {
		return err
	}

	port := strconv.Itoa(clientPort)

	writeLine(natRules,
		"-A", string(KUBERNIKUS_TUNNELS),
		"-m", "comment", "--comment", fmt.Sprintf(`"node ip tunnel redirect for %s"`, key),
		"--dst", ip.String(),
		"-p", "tcp",
		"-j", "REDIRECT",
		"--to-ports", port,
	)

	writeLine(natRules,
		"-A", string(KUBERNIKUS_TUNNELS),
		"-m", "comment", "--comment", fmt.Sprintf(`"pod cidr tunnel redirect for %s"`, key),
		"--dst", node.Spec.PodCIDR,
		"-p", "tcp",
		"-j", "REDIRECT",
		"--to-ports", port,
	)

	// Write one service CIDR iptable for each node.
	// Only the first one will be used though, so the first node will take all service traffic.
	writeLine(natRules,
		"-A", string(KUBERNIKUS_TUNNELS),
		"-m", "comment", "--comment", `"cluster service CIDR tunnel redirect"`,
		"--dst", c.serviceCIDR,
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
		fmt.Printf("node ip is : %s", addresses[0].Address)
		return net.ParseIP(addresses[0].Address), nil
	}
	if addresses, ok := addressMap[v1.NodeExternalIP]; ok {
		fmt.Printf("node ip is : %s", addresses[0].Address)
		return net.ParseIP(addresses[0].Address), nil
	}
	return nil, fmt.Errorf("host IP unknown; known addresses: %v", addresses)
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
