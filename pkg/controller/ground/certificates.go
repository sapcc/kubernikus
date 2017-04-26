package ground

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"time"

	certutil "k8s.io/client-go/util/cert"
)

const (
	duration365d = time.Hour * 24 * 365
)

type ClusterCerts struct {
	Name       string
	Etcd       *EtcdCerts
	Kubernetes *KubernetesCerts
}

type EtcdCerts struct {
	Peers   *Pool
	Clients *Pool
}

type KubernetesCerts struct {
	Clients  *Pool
	Kubelets *Pool
	TLS      *Pool
}

type Pool struct {
	CA           *Bundle
	Certificates []*Bundle
}

type Bundle struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

type Config struct {
	CommonName         string
	Organization       []string
	OrganizationalUnit []string
	Locality           []string
	Province           []string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
}

type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

func newClusterCerts(name string) ClusterCerts {
	dict := ClusterCerts{
		Name: name,
	}

	dict.Etcd = dict.initializeEtcdCerts()
	dict.Kubernetes = dict.initializeKubernetesCerts()

	return dict
}

func (dict ClusterCerts) initializeEtcdCerts() *EtcdCerts {
	return &EtcdCerts{
		Peers:   dict.newPool("Etcd Peers"),
		Clients: dict.newPool("Etcd Clients"),
	}
}

func (dict ClusterCerts) initializeKubernetesCerts() *KubernetesCerts {
	return &KubernetesCerts{
		Clients:  dict.newPool("Kubernetes Clients"),
		Kubelets: dict.newPool("Kubernetes Kubelets"),
		TLS:      dict.newPool("Kubernetes TLS"),
	}
}

func (c ClusterCerts) bundles() []*Bundle {
	bundles := []*Bundle{}
	bundles = append(bundles, c.Etcd.Clients.bundles()...)
	bundles = append(bundles, c.Etcd.Peers.bundles()...)
	bundles = append(bundles, c.Kubernetes.Clients.bundles()...)
	bundles = append(bundles, c.Kubernetes.Kubelets.bundles()...)
	bundles = append(bundles, c.Kubernetes.TLS.bundles()...)

	return bundles
}

func (p Pool) bundles() []*Bundle {
	return append(p.Certificates, p.CA)
}

func (dict ClusterCerts) newPool(name string) *Pool {
	ca, err := dict.newCA(name)
	if err != nil {
		panic(err)
	}
	pool := &Pool{CA: ca}
	return pool
}

func (dict ClusterCerts) newCA(name string) (*Bundle, error) {
	key, err := certutil.NewPrivateKey()
	if err != nil {
		panic(err)
		return &Bundle{}, fmt.Errorf("unable to create private key [%v]", err)
	}

	config := Config{
		OrganizationalUnit: []string{"SAP Converged Cloud", "Kubernikus"},
		Province:           []string{fmt.Sprintf("%s CA", name)},
		Locality:           []string{dict.Name},
	}
	cert, err := NewSelfSignedCACert(config, key)

	if err != nil {
		panic(err)
		return &Bundle{}, fmt.Errorf("unable to create self-signed certificate [%v]", err)
	}

	return &Bundle{cert, key}, nil
}

func NewSelfSignedCACert(cfg Config, key *rsa.PrivateKey) (*x509.Certificate, error) {
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:         cfg.CommonName,
			Organization:       cfg.Organization,
			OrganizationalUnit: cfg.OrganizationalUnit,
			Province:           cfg.Province,
			Locality:           cfg.Locality,
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d * 10).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}

	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func NewSignedCert(cfg Config, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:         cfg.CommonName,
			Organization:       cfg.Organization,
			OrganizationalUnit: cfg.OrganizationalUnit,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}
