package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"net"

	"github.com/golang/glog"
	"github.com/koding/tunnel"
)

var (
	ca     string
	cert   string
	key    string
	server string
)

func main() {

	flag.StringVar(&ca, "ca", "", "Server CA")
	flag.StringVar(&cert, "cert", "", "Server CA")
	flag.StringVar(&key, "key", "", "Server CA")
	flag.StringVar(&server, "server", "", "Server CA")
	flag.Parse()

	var rootCAs *x509.CertPool
	if ca != "" {
		rootCAs = x509.NewCertPool()
		content, err := ioutil.ReadFile(ca)
		if err != nil {
			glog.Fatalf("Failed to to read file %s: %s", ca, err)
		}
		if !rootCAs.AppendCertsFromPEM(content) {
			glog.Fatalf("Failed to load any certs from %s", ca)
		}
	}
	certificate, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		glog.Fatalf("Failed to load certificate/key: %s", err)
	}

	x509cert, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		glog.Fatalf("Failed to extract common name from client cert: %s", err)
	}

	cfg := &tunnel.ClientConfig{
		Identifier: x509cert.Subject.CommonName,
		ServerAddr: server,
		Dial: func(network, address string) (net.Conn, error) {
			return tls.Dial(network, address, &tls.Config{
				RootCAs:      rootCAs,
				Certificates: []tls.Certificate{certificate},
			})
		},
	}

	client, err := tunnel.NewClient(cfg)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof("Connecting to %s with id %s", cfg.ServerAddr, cfg.Identifier)

	client.Start()
}
