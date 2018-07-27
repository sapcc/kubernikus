package wormhole

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

func GetTLSConfig(certificatePath string, privateKeyPath string, clientCAPath string) (*tls.Config, error) {
	tlsConfig, err := NewTLSConfig(certificatePath, privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load cert or key: %s", err)
	}
	caPool, err := LoadCAFile(clientCAPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load ca file %s: %s", clientCAPath, err)
	}
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	tlsConfig.ClientCAs = caPool

	return tlsConfig, nil
}

func LoadCAFile(cafile string) (*x509.CertPool, error) {
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

func NewTLSConfig(cert, key string) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}, nil
}
