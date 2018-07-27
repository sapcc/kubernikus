package server

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/databus23/guttle"
	"github.com/go-kit/kit/log"

	"github.com/sapcc/kubernikus/pkg/wormhole"
)

type Options struct {
	Logger      log.Logger
	KubeConfig  string
	Context     string
	ClientCA    string
	Certificate string
	PrivateKey  string
	ServiceCIDR string
}

func New(options *Options) (*guttle.Server, error) {
	var listener net.Listener
	var err error

	// Configure TLS
	tlsConfig, err := wormhole.NewTLSConfig(options.Certificate, options.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to load cert or key: %s", err)
	}
	tlsConfig.ClientCAs, err = wormhole.LoadCAFile(options.ClientCA)
	if err != nil {
		return nil, fmt.Errorf("Failed to load ca file %s: %s", options.ClientCA, err)
	}
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	listener, err = tls.Listen("tcp", "0.0.0.0:9090", tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen to 0.0.0.0:9090: %s", err)
	}
	fmt.Printf("Listenning on 0.0.0.0:9090 with TLS")

	opts := guttle.ServerOptions{
		Listener:  listener,
		ProxyFunc: guttle.SourceRoutedProxy(),
	}

	return guttle.NewServer(&opts), nil
}
