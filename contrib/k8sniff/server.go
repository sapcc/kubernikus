/* {{{ Copyright (c) Paul R. Tagliamonte <paultag@debian.org>, 2015
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE. }}} */

package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/k8sniff/metrics"
	"github.com/kubermatic/k8sniff/parser"

	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// ingressClassKey picks a specific "class" for the Ingress. The controller
	// only processes Ingresses with this annotation either unset, or set
	// to either nginxIngressClass or the empty string.
	ingressClassKey = "kubernetes.io/ingress.class"

	ConnectionClosedErr = "use of closed network connection"
	ConnectionResetErr  = "connection reset by peer"
)

// now provides func() time.Time
// so it is easier to mock, if wou want to add tests
var now = time.Now

type ServerAndRegexp struct {
	Server *Server
	Regexp *regexp.Regexp
}

type Proxy struct {
	Lock       sync.RWMutex
	ServerList []ServerAndRegexp
	Default    *Server
}

func (p *Proxy) Get(host string) *Server {
	p.Lock.RLock()
	defer p.Lock.RUnlock()

	for _, tuple := range p.ServerList {
		if tuple.Regexp.MatchString(host) {
			return tuple.Server
		}
	}
	return p.Default
}

func (p *Proxy) Update(c *Config) error {
	servers := []ServerAndRegexp{}
	currentServers := c.Servers
	for i, server := range currentServers {
		for _, hostname := range server.Names {
			var hostRegexp *regexp.Regexp
			var err error
			if server.Regexp {
				hostRegexp, err = regexp.Compile(hostname)
			} else {
				hostRegexp, err = regexp.Compile("^" + regexp.QuoteMeta(hostname) + "$")
			}
			if err != nil {
				return fmt.Errorf("cannot update proxy due to invalid regex: %v", err)
			}
			tuple := ServerAndRegexp{&currentServers[i], hostRegexp}
			servers = append(servers, tuple)
		}
	}
	var def *Server
	for i, server := range currentServers {
		if server.Default {
			def = &currentServers[i]
			break
		}
	}

	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.ServerList = servers
	p.Default = def

	return nil
}

func (c *Config) UpdateServers() error {
	class := c.Kubernetes.IngressClass
	if class == "" {
		class = "k8sniff"
	}

	serverForBackend := func(ing *netv1.Ingress, backend *netv1.IngressBackend) (*Server, error) {
		obj, found, err := c.serviceStore.GetByKey(fmt.Sprintf("%s/%s", ing.Namespace, backend.Service.Name))
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("service %s/%s not found", ing.Namespace, backend.Service.Name)
		}
		svc := obj.(*v1.Service)
		port := int(backend.Service.Port.Number)
		return &Server{
			Host: svc.Spec.ClusterIP,
			Port: port,
		}, nil
	}

	servers := []Server{}
	ingressList := c.ingressStore.List()
	for _, i := range ingressList {
		i := i.(*netv1.Ingress)
		name := fmt.Sprintf("%s/%s", i.Namespace, i.Name)
		if i.Annotations[ingressClassKey] != class {
			glog.V(6).Infof("Skipping ingress %s due to missing annotation. Expected %s=%s Got %s=%s", name, ingressClassKey, class, ingressClassKey, i.Annotations[ingressClassKey])
			continue
		}

		if i.Spec.DefaultBackend != nil {
			s, err := serverForBackend(i, i.Spec.DefaultBackend)
			if err != nil {
				metrics.IncErrors(metrics.Error)
				glog.V(0).Infof("Ingress %s error with default backend, skipping: %v", name, err)
			} else {
				s.Default = true
				servers = append(servers, *s)
			}
		}
		for _, r := range i.Spec.Rules {
			if r.HTTP == nil {
				metrics.IncErrors(metrics.Error)
				glog.V(0).Infof("Ingress %s error with rule, skipping: http must be set", name)
				continue
			}
			for _, p := range r.HTTP.Paths {
				if p.Path != "" && p.Path != "/" {
					metrics.IncErrors(metrics.Error)
					glog.V(0).Infof("Ingress %s error with rule, skipping: path is not empty", name)
					continue
				}
				s, err := serverForBackend(i, &p.Backend)
				if err != nil {
					metrics.IncErrors(metrics.Error)
					glog.V(0).Infof("Ingress %s error with rule %q path %q, skipping: %v", name, r.Host, p.Path, err)
					continue
				}
				s.Names = []string{r.Host}
				glog.V(6).Infof("Adding backend %q -> %s:%d", r.Host, s.Host, s.Port)
				servers = append(servers, *s)
			}
		}
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if !reflect.DeepEqual(c.Servers, servers) {
		c.Servers = servers
		glog.V(2).Infof("Updating proxy configuration")
		err := c.proxy.Update(c)
		if err != nil {
			time.Sleep(time.Second)
			return fmt.Errorf("failed to update proxy: %v", err)
		}
		glog.V(2).Infof("================================================")
		glog.V(2).Infof("Updated servers. New servers:")
		c.PrintCurrentServers(2)
		glog.V(2).Infof("================================================")
	}

	metrics.SetBackendCount(len(c.Servers) - 1)

	return nil
}

func (c *Config) PrintCurrentServers(logLevel glog.Level) {
	for _, s := range c.Servers {
		hostnames := strings.Join(s.Names, ",")
		if hostnames == "" {
			hostnames = "default backend"
		}
		glog.V(logLevel).Infof("%s -> %s", hostnames, s.Host)
	}
}

func (c *Config) Debug() {
	glog.V(4).Info("================================================")
	glog.V(4).Info("Current configured servers:")
	c.PrintCurrentServers(4)
	glog.V(4).Info("================================================")
}

func (c *Config) TriggerUpdate() {
	if !c.ControllersHaveSynced() {
		return
	}
	err := c.UpdateServers()
	if err != nil {
		metrics.IncErrors(metrics.Info)
		glog.V(0).Infof("failed to update servers list: %v", err)
	}
}

func (c *Config) ControllersHaveSynced() bool {
	return c.ingressController.HasSynced() && c.serviceController.HasSynced()
}

func (c *Config) Serve(stopCh chan struct{}) error {
	glog.V(0).Infof("Listening on %s:%d", c.Bind.Host, c.Bind.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf(
		"%s:%d", c.Bind.Host, c.Bind.Port,
	))
	if err != nil {
		metrics.IncErrors(metrics.Fatal)
		return err
	}

	c.proxy = &Proxy{}
	err = c.proxy.Update(c)
	if err != nil {
		metrics.IncErrors(metrics.Fatal)
		return err
	}

	if c.Kubernetes != nil {
		cfg, err := clientcmd.BuildConfigFromFlags("", c.Kubernetes.Kubeconfig)
		if err != nil {
			panic(err)
		}
		c.Kubernetes.Client = kubernetes.NewForConfigOrDie(cfg)
		c.ingressStore, c.ingressController = cache.NewInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return c.Kubernetes.Client.NetworkingV1().Ingresses("").List(context.TODO(), options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return c.Kubernetes.Client.NetworkingV1().Ingresses("").Watch(context.TODO(), options)
				},
			},
			&netv1.Ingress{},
			30*time.Minute,
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					go c.TriggerUpdate()
				},
				UpdateFunc: func(old, cur interface{}) {
					go c.TriggerUpdate()
				},
				DeleteFunc: func(obj interface{}) {
					go c.TriggerUpdate()
				},
			},
		)

		c.serviceStore, c.serviceController = cache.NewInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return c.Kubernetes.Client.CoreV1().Services("").List(context.TODO(), options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return c.Kubernetes.Client.CoreV1().Services("").Watch(context.TODO(), options)
				},
			},
			&v1.Service{},
			30*time.Minute,
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					go c.TriggerUpdate()
				},
				UpdateFunc: func(old, cur interface{}) {
					go c.TriggerUpdate()
				},
				DeleteFunc: func(obj interface{}) {
					go c.TriggerUpdate()
				},
			},
		)

		go c.serviceController.Run(stopCh)
		go c.ingressController.Run(stopCh)
	}
	c.TriggerUpdate()

	go wait.Forever(func() {
		c.Debug()
	}, 30*time.Second)

	for {
		conn, err := listener.Accept()
		if err != nil {
			metrics.IncErrors(metrics.Error)
			return err
		}

		connectionID := RandomString(8)
		glog.V(4).Infof(
			"[%s] Proxy: %s -> %s",
			connectionID,
			conn.RemoteAddr(),
			conn.LocalAddr(),
		)
		go c.proxy.Handle(conn, connectionID)
	}
}

func (p *Proxy) Handle(conn net.Conn, connectionID string) {
	metrics.IncConnections()
	start := now()
	defer func(s time.Time) {
		err := conn.Close()
		if err != nil {
			glog.V(0).Infof("[%s] Failed closing connection: %v", connectionID, err)
			metrics.IncErrors(metrics.Error)
		}
		metrics.DecConnections()
		metrics.ConnectionTime(now().Sub(s))
	}(start)
	data := make([]byte, 4096)

	length, err := conn.Read(data)
	if err != nil {
		metrics.IncErrors(metrics.Error)
		glog.V(4).Infof("[%s] Error reading the first 4k of the connection: %v", connectionID, err)
		return
	}

	var proxy *Server
	hostname, hostnameErr := parser.GetHostname(data[:])
	if hostnameErr == nil {
		glog.V(6).Infof("[%s] Parsed hostname: %s", connectionID, hostname)

		proxy = p.Get(hostname)
		if proxy == nil {
			glog.V(4).Infof("[%s] No proxy matched %s", connectionID, hostname)
			return
		} else {
			glog.V(4).Infof("[%s] Host found %s", connectionID, proxy.Host)
		}
	} else {
		glog.V(6).Infof("[%s] Parsed request without hostname", connectionID)

		proxy = p.Default
		if proxy == nil {
			glog.V(4).Infof("[%s] No default proxy", connectionID)
			return
		}
	}

	clientConn, err := net.Dial("tcp", fmt.Sprintf(
		"%s:%d", proxy.Host, proxy.Port,
	))
	if err != nil {
		metrics.IncErrors(metrics.Error)
		glog.V(0).Infof("[%s] Error connecting to backend: %v", connectionID, err)
		return
	}

	defer func() {
		err := clientConn.Close()
		if err != nil {
			glog.V(0).Infof("[%s] Failed closing client connection: %v", connectionID, err)
			metrics.IncErrors(metrics.Error)
		}
	}()

	n, err := clientConn.Write(data[:length])
	glog.V(7).Infof("[%s] Wrote %d bytes", connectionID, n)
	if err != nil {
		metrics.IncErrors(metrics.Info)
		glog.V(7).Infof("[%s] Error sending data to backend: %v", connectionID, err)
		return
	}
	Copycat(clientConn, conn, connectionID)
}

func Copycat(client, server net.Conn, connectionID string) {
	glog.V(6).Infof("[%s] Initiating copy between %s and %s", connectionID, client.RemoteAddr().String(), server.RemoteAddr().String())

	doCopy := func(s, c net.Conn, cancel chan<- bool) {
		glog.V(7).Infof("[%s] Established connection %s -> %s", connectionID, s.RemoteAddr().String(), c.RemoteAddr().String())
		_, err := io.Copy(s, c)
		if err != nil && !strings.Contains(err.Error(), ConnectionClosedErr) && !strings.Contains(err.Error(), ConnectionResetErr) {
			glog.V(0).Infof("[%s] Failed copying connection data: %v", connectionID, err)
			metrics.IncErrors(metrics.Error)
		}
		glog.V(7).Infof("[%s] Destroyed connection %s -> %s", connectionID, s.RemoteAddr().String(), c.RemoteAddr().String())
		cancel <- true
	}

	cancel := make(chan bool, 2)

	go doCopy(server, client, cancel)
	go doCopy(client, server, cancel)

	select {
	case <-cancel:
		glog.V(6).Infof("[%s] Disconnected", connectionID)
		return
	}
}

func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
