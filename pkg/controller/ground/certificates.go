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

type Certificates struct {
	Etcd struct {
		Clients struct {
			CA        *Bundle
			Apiserver *Bundle
		}
	}

	Kubernetes struct {
		Clients struct {
			CA                *Bundle
			ControllerManager *Bundle
			Scheduler         *Bundle
			Proxy             *Bundle
			Kubelet           *Bundle
			ClusterAdmin      *Bundle
		}
		Nodes struct {
			CA      *Bundle
			Generic *Bundle
		}
	}

	TLS struct {
		CA        *Bundle
		ApiServer *Bundle
		Etcd      *Bundle
	}
}

type Bundle struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

type Config struct {
	CommonName         string
	Organization       []string
	OrganizationalUnit []string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
}

type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

func (c Certificates) all() []*Bundle {
	return []*Bundle{
		c.Etcd.Clients.Apiserver,
		c.Etcd.Clients.CA,
		c.Kubernetes.Clients.CA,
		c.Kubernetes.Clients.ControllerManager,
		c.Kubernetes.Clients.Scheduler,
		c.Kubernetes.Clients.Proxy,
		c.Kubernetes.Clients.Kubelet,
		c.Kubernetes.Clients.ClusterAdmin,
		c.Kubernetes.Nodes.CA,
		c.Kubernetes.Nodes.Generic,
		c.TLS.CA,
		c.TLS.ApiServer,
		c.TLS.Etcd,
	}
}

func newCertificates(satellite string) (*Certificates, error) {
	certs := &Certificates{}

	if ca, err := newCA(satellite, "Etcd Clients"); err != nil {
		return certs, err
	} else {
		certs.Etcd.Clients.CA = ca
	}

	if ca, err := newCA(satellite, "Kubernetes Clients"); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Clients.CA = ca
	}

	if ca, err := newCA(satellite, "Kubernetes Nodes"); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Nodes.CA = ca
	}

	if ca, err := newCA(satellite, "TLS"); err != nil {
		return certs, err
	} else {
		certs.TLS.CA = ca
	}

	if cert, err := newSignedBundle(Config{
		CommonName: "apiserver",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, certs.Etcd.Clients.CA); err != nil {
		return certs, err
	} else {
		certs.Etcd.Clients.Apiserver = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName:   "cluster-admin",
		Organization: []string{"system:masters"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, certs.Kubernetes.Clients.CA); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Clients.ClusterAdmin = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName: "system:kube-controller-manager",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, certs.Kubernetes.Clients.CA); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Clients.ControllerManager = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName:   "kuelet",
		Organization: []string{"system:nodes"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, certs.Kubernetes.Clients.CA); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Clients.Kubelet = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName: "system:kube-proxy",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, certs.Kubernetes.Clients.CA); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Clients.Proxy = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName: "system:kube-scheduler",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, certs.Kubernetes.Clients.CA); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Clients.Scheduler = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName:   "kubelet",
		Organization: []string{"system:nodes"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, certs.Kubernetes.Nodes.CA); err != nil {
		return certs, err
	} else {
		certs.Kubernetes.Nodes.Generic = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName: "apiserver",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames: AltNames{
			DNSNames: []string{"kubernetes", "kubernetes.default", "apiserver", "TODO:external.dns.name"},
			IPs:      []net.IP{net.IPv4(127, 0, 0, 1)},
		},
	}, certs.TLS.CA); err != nil {
		return certs, err
	} else {
		certs.TLS.ApiServer = cert
	}

	if cert, err := newSignedBundle(Config{
		CommonName: "etcd",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames: AltNames{
			DNSNames: []string{"etcd"},
			IPs:      []net.IP{net.IPv4(127, 0, 0, 1)},
		},
	}, certs.TLS.CA); err != nil {
		return certs, err
	} else {
		certs.TLS.Etcd = cert
	}

	return certs, nil
}

func newCA(satellite, name string) (*Bundle, error) {
	key, err := certutil.NewPrivateKey()
	if err != nil {
		panic(err)
		return &Bundle{}, fmt.Errorf("unable to create private key [%v]", err)
	}

	config := Config{
		CommonName:         name,
		OrganizationalUnit: []string{"SAP Converged Cloud", "Kubernikus", satellite},
	}
	cert, err := NewSelfSignedCACert(config, key)

	if err != nil {
		panic(err)
		return &Bundle{}, fmt.Errorf("unable to create self-signed certificate [%v]", err)
	}

	return &Bundle{cert, key}, nil
}

func newSignedBundle(config Config, ca *Bundle) (*Bundle, error) {
	key, err := certutil.NewPrivateKey()
	if err != nil {
		return &Bundle{}, fmt.Errorf("unable to create private key [%v]", err)
	}

	config.OrganizationalUnit = ca.Certificate.Subject.OrganizationalUnit

	cert, err := NewSignedCert(config, key, ca.Certificate, ca.PrivateKey)

	if err != nil {
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
		NotAfter:     time.Now().Add(duration365d * 10).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}
