package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"sync"

	"github.com/databus23/guttle"
	"github.com/go-kit/kit/log"
)

type Tunnel struct {
	Server *guttle.Server

	Logger log.Logger
}

func NewTunnel(options *Options) (*Tunnel, error) {
	logger := log.With(options.Logger, "wormhole", "tunnel")

	var listener net.Listener
	if options.Certificate != "" {
		tlsConfig, err := newTLSConfig(options.Certificate, options.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("Failed to load cert or key: %s", err)
		}
		caPool, err := loadCAFile(options.ClientCA)
		if err != nil {
			return nil, fmt.Errorf("Failed to load ca file %s: %s", options.ClientCA, err)
		}
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = caPool
		listener, err = tls.Listen("tcp", "0.0.0.0:6553", tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("Failed to listen to 0.0.0.0:6553: %s", err)
		}
	} else {
		var err error
		listener, err = net.Listen("tcp", "127.0.0.1:8080")
		if err != nil {
			return nil, fmt.Errorf("Failed to listen to 127.0.0.1:8080: %s", err)
		}
	}
	logger.Log(
		"msg", "Listening for tunnel clients",
		"addr", listener.Addr())

	opts := guttle.ServerOptions{
		Listener:   listener,
		HijackAddr: "127.0.0.1:9191",
		ProxyFunc:  guttle.StaticProxy("127.0.0.1:6443"),
	}

	return &Tunnel{Server: guttle.NewServer(&opts), Logger: logger}, nil
}

func (t *Tunnel) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	go func() {
		<-stopCh
		t.Server.Close()
	}()

	err := t.Server.Start()
	if err != nil {
		t.Logger.Log("err", err)
	}

}

func loadCAFile(cafile string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	cas, err := ioutil.ReadFile(cafile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %s", cafile, err)
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
