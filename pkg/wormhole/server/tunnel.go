package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/databus23/guttle"
	"github.com/golang/glog"
)

type Tunnel struct {
	Server  *guttle.Server
	options *TunnelOptions
}

type TunnelOptions struct {
	ClientCA    string
	Certificate string
	PrivateKey  string
}

func NewTunnel(options *TunnelOptions) *Tunnel {
	return &Tunnel{options: options}
}

func (t *Tunnel) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	caPool, err := loadCAFile(t.options.ClientCA)
	if err != nil {
		glog.Info(err)
		return
	}

	tlsConfig, err := newTLSConfig(t.options.Certificate, t.options.PrivateKey)
	if err != nil {
		glog.Info(err)
		return
	}

	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	tlsConfig.RootCAs = caPool

	listener, err := tls.Listen("tcp", "0.0.0.0:443", tlsConfig)
	if err != nil {
		glog.Error(err)
		return
	}

	opts := guttle.ServerOptions{
		Listener:   listener,
		HijackAddr: "127.0.0.1:9191",
		ProxyFunc:  guttle.StaticProxy("127.0.0.1:6443"),
	}

	t.Server = guttle.NewServer(&opts)

	go func() {
		<-stopCh
		t.Server.Close()
	}()

	err = t.Server.Start()
	if err != nil {
		glog.Error(err)
	}

}

func loadCAFile(cafile string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	cas, err := ioutil.ReadFile(cafile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %s", cafile)
	}
	if !pool.AppendCertsFromPEM(cas) {
		return nil, fmt.Errorf("No certs found in file %s", cafile)
	}
	return pool, nil
}

func newTLSConfig(cert, key string) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}, nil
}
