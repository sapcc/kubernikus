package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

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
