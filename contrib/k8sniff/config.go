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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	maxTCPPort = 65535
)

type Config struct {
	Bind       Bind
	Metrics    Metrics
	Servers    []Server
	Kubernetes *Kubernetes
	proxy      *Proxy
	lock       sync.Mutex

	serviceController cache.Controller
	serviceStore      cache.Store
	ingressController cache.Controller
	ingressStore      cache.Store
}

// Valid returns an if the config is invalid
func (c *Config) Valid() error {
	if len(c.Servers) > 0 && c.Kubernetes != nil {
		return errors.New("Cannot set .Servers and .Kubernetes in config file")
	}

	if err := c.Metrics.Valid(); err != nil {
		return err
	}

	return nil
}

type Kubernetes struct {
	Kubeconfig   string
	Client       *kubernetes.Clientset
	IngressClass string
}

// Metrics contains the port & path for the
// prometheus endpoint
type Metrics struct {
	Host string
	Port int
	Path string
}

type Bind struct {
	Host string
	Port int
}

type Server struct {
	Default bool
	Regexp  bool
	Host    string
	Names   []string
	Port    int
}

// Valid returns an error if the metrics config is invalid
func (m Metrics) Valid() error {
	if m.Port > maxTCPPort {
		return fmt.Errorf("Configured metrics port is above %d: port=%d", maxTCPPort, m.Port)
	}

	return nil
}

func LoadConfig(path string) (*Config, error) {
	glog.V(5).Infof("Loading config from: %s", path)

	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	config := Config{}
	err = json.NewDecoder(fd).Decode(&config)
	if err != nil {
		return nil, err
	}

	if err = config.Valid(); err != nil {
		return nil, err
	}

	config.setDefaultsIfUnset()

	glog.V(5).Infof("Read config: %v", config)

	return &config, err
}

func (c *Config) setDefaultsIfUnset() {
	if c.Bind.Port == 0 {
		c.Bind.Port = 8443
	}

	if c.Bind.Host == "" {
		glog.V(5).Infof("Bind host not set. Using default: 0.0.0.0")
		c.Bind.Host = "0.0.0.0"
	}

	if c.Metrics.Host == "" {
		glog.V(5).Infof("Metrics host not set. Using default: 0.0.0.0")
		c.Metrics.Host = c.Bind.Host
	}

	if c.Metrics.Port == 0 {
		glog.V(5).Infof("Metrics port not set. Using default: 9091")
		c.Metrics.Port = 9091
	}

	if c.Metrics.Path == "" {
		glog.V(5).Infof("Metrics path not set. Using default: /metrics")
		c.Metrics.Path = "/metrics"
	}
	if !strings.HasPrefix(c.Metrics.Path, "/") {
		c.Metrics.Path = "/" + c.Metrics.Path
	}

}
