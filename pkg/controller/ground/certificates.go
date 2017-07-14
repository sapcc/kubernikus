package ground

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/kennygrant/sanitize"

	certutil "k8s.io/client-go/util/cert"
)

const (
	duration365d = time.Hour * 24 * 365
)

type Certificates struct {
	Etcd struct {
		Clients struct {
			CA        Bundle
			ApiServer Bundle
		}
		Peers struct {
			CA        Bundle
			Universal Bundle
		}
	}

	ApiServer struct {
		Clients struct {
			CA                Bundle
			ControllerManager Bundle
			Scheduler         Bundle
			Proxy             Bundle
			ClusterAdmin      Bundle
		}
		Nodes struct {
			CA        Bundle
			Universal Bundle
		}
	}

	Kubelet struct {
		Clients struct {
			CA        Bundle
			ApiServer Bundle
		}
	}

	TLS struct {
		CA        Bundle
		ApiServer Bundle
	}
}

func (c *Certificates) MarshalYAML() (interface{}, error) {
	bundles := c.all()
	result := make(map[string]string, len(bundles)*2)
	for _, bundle := range bundles {
		result[bundle.NameForCert()] = string(certutil.EncodeCertPEM(bundle.Certificate))
		result[bundle.NameForKey()] = string(certutil.EncodePrivateKeyPEM(bundle.PrivateKey))
	}

	return result, nil
}

type Bundle struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

func (b *Bundle) basename() string {
	stem := ""
	suffix := ""

	if b.Certificate.IsCA {
		stem = b.Certificate.Subject.CommonName
		suffix = "ca"
	} else {
		stem = b.Certificate.Issuer.CommonName
		suffix = b.Certificate.Subject.CommonName
	}

	return sanitize.BaseName(strings.ToLower(fmt.Sprintf("%s-%s", stem, suffix)))
}

func (b *Bundle) NameForKey() string {
	return fmt.Sprintf("%s-key.pem", b.basename())
}
func (b *Bundle) NameForCert() string {
	return fmt.Sprintf("%s.pem", b.basename())
}

type Config struct {
	sign               string
	Organization       []string
	OrganizationalUnit []string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
}

type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

func (c Certificates) all() []Bundle {
	return []Bundle{
		c.Etcd.Clients.CA,
		c.Etcd.Clients.ApiServer,
		c.Etcd.Peers.CA,
		c.Etcd.Peers.Universal,
		c.ApiServer.Clients.CA,
		c.ApiServer.Clients.ControllerManager,
		c.ApiServer.Clients.Scheduler,
		c.ApiServer.Clients.Proxy,
		c.ApiServer.Clients.ClusterAdmin,
		c.ApiServer.Nodes.CA,
		c.ApiServer.Nodes.Universal,
		c.Kubelet.Clients.CA,
		c.Kubelet.Clients.ApiServer,
		c.TLS.CA,
		c.TLS.ApiServer,
	}
}

func (certs *Certificates) populateForSatellite(satellite string) error {
	createCA(satellite, "Etcd Clients", &certs.Etcd.Clients.CA)
	createCA(satellite, "Etcd Peers", &certs.Etcd.Peers.CA)
	createCA(satellite, "ApiServer Clients", &certs.ApiServer.Clients.CA)
	createCA(satellite, "ApiServer Nodes", &certs.ApiServer.Nodes.CA)
	createCA(satellite, "Kubelet Clients", &certs.Kubelet.Clients.CA)
	createCA(satellite, "TLS", &certs.TLS.CA)

	certs.Etcd.Clients.ApiServer = certs.signEtcdClient("apiserver")
	certs.Etcd.Peers.Universal = certs.signEtcdPeer("universal")
	certs.ApiServer.Clients.ClusterAdmin = certs.signApiServerClient("cluster-admin", "system:masters")
	certs.ApiServer.Clients.ControllerManager = certs.signApiServerClient("system:kube-controller-manager")
	certs.ApiServer.Clients.Proxy = certs.signApiServerClient("system:kube-proxy")
	certs.ApiServer.Clients.Scheduler = certs.signApiServerClient("system:kube-scheduler")
	certs.ApiServer.Nodes.Universal = certs.signApiServerNode("universal")
	certs.Kubelet.Clients.ApiServer = certs.signKubeletClient("apiserver")
	certs.TLS.ApiServer = certs.signTLS("apiserver",
		[]string{"kubernetes", "kubernetes.default", "apiserver", "TODO:external.dns.name"},
		[]net.IP{net.IPv4(127, 0, 0, 1)})

	return nil
}

func (c Certificates) signEtcdClient(name string) Bundle {
	config := Config{
		sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.Etcd.Clients.CA.sign(config)
}

func (c Certificates) signEtcdPeer(name string) Bundle {
	config := Config{
		sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	return c.Etcd.Peers.CA.sign(config)
}

func (c Certificates) signApiServerClient(name string, groups ...string) Bundle {
	config := Config{
		sign:         name,
		Organization: groups,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.ApiServer.Clients.CA.sign(config)
}

func (c Certificates) signApiServerNode(name string) Bundle {
	config := Config{
		sign:         name,
		Organization: []string{"system:nodes"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.ApiServer.Nodes.CA.sign(config)
}

func (c Certificates) signKubeletClient(name string) Bundle {
	config := Config{
		sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.Kubelet.Clients.CA.sign(config)
}

func (c Certificates) signTLS(name string, dnsNames []string, ips []net.IP) Bundle {
	config := Config{
		sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames: AltNames{
			DNSNames: dnsNames,
			IPs:      ips,
		},
	}
	return c.TLS.CA.sign(config)
}

func createCA(satellite, name string, bundle *Bundle) {
	bundle.PrivateKey, _ = certutil.NewPrivateKey()

	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:         name,
			OrganizationalUnit: []string{"SAP Converged Cloud", "Kubernikus", satellite},
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d * 10).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}

	certDERBytes, _ := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, bundle.PrivateKey.Public(), bundle.PrivateKey)
	bundle.Certificate, _ = x509.ParseCertificate(certDERBytes)
}

func (ca Bundle) sign(config Config) Bundle {
	if !ca.Certificate.IsCA {
		panic("You can't use this certificate for signing. It's not a CA...")
	}

	key, _ := certutil.NewPrivateKey()
	serial, _ := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:         config.sign,
			Organization:       config.Organization,
			OrganizationalUnit: ca.Certificate.Subject.OrganizationalUnit,
		},
		DNSNames:     config.AltNames.DNSNames,
		IPAddresses:  config.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    ca.Certificate.NotBefore,
		NotAfter:     time.Now().Add(duration365d * 10).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  config.Usages,
	}

	certDERBytes, _ := x509.CreateCertificate(rand.Reader, &certTmpl, ca.Certificate, key.Public(), ca.PrivateKey)

	cert, _ := x509.ParseCertificate(certDERBytes)

	return Bundle{cert, key}
}
