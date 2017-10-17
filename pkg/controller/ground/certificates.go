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
	"strings"
	"time"

	"github.com/kennygrant/sanitize"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"

	certutil "k8s.io/client-go/util/cert"
)

const (
	//by default we generate certs with 2 year validity
	defaultCertValidity = 2 * time.Hour * 24 * 365
	//out CAs are valid for 10 years
	caValidity = 10 * time.Hour * 24 * 365
)

type Bundle struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

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
			Wormhole          Bundle
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
		Wormhole  Bundle
	}
}

func (c *Certificates) toMap() map[string]string {
	bundles := c.all()
	result := make(map[string]string, len(bundles)*2)
	for _, bundle := range bundles {
		result[bundle.NameForCert()] = string(certutil.EncodeCertPEM(bundle.Certificate))
		result[bundle.NameForKey()] = string(certutil.EncodePrivateKeyPEM(bundle.PrivateKey))
	}
	return result
}

func (c *Certificates) MarshalYAML() (interface{}, error) {
	return c.toMap(), nil
}

func NewBundle(key, cert []byte) (Bundle, error) {
	certificates, err := certutil.ParseCertsPEM(cert)
	if err != nil {
		return Bundle{}, err
	}
	if len(certificates) < 1 {
		return Bundle{}, errors.New("No certificates found")
	}
	k, err := certutil.ParsePrivateKeyPEM(key)
	if err != nil {
		return Bundle{}, err
	}
	rsaKey, isRSAKey := k.(*rsa.PrivateKey)
	if !isRSAKey {
		return Bundle{}, errors.New("Key does not seem to be of type RSA")
	}

	return Bundle{PrivateKey: rsaKey, Certificate: certificates[0]}, nil
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
	Sign               string
	Organization       []string
	OrganizationalUnit []string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
	ValidFor           time.Duration
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
		c.ApiServer.Clients.Wormhole,
		c.ApiServer.Nodes.CA,
		c.ApiServer.Nodes.Universal,
		c.Kubelet.Clients.CA,
		c.Kubelet.Clients.ApiServer,
		c.TLS.CA,
		c.TLS.ApiServer,
		c.TLS.Wormhole,
	}
}

func CreateCertificates(kluster *v1.Kluster, domain string) {
	certs := &Certificates{}
	createCA(kluster.Name, "Etcd Clients", &certs.Etcd.Clients.CA)
	createCA(kluster.Name, "Etcd Peers", &certs.Etcd.Peers.CA)
	createCA(kluster.Name, "ApiServer Clients", &certs.ApiServer.Clients.CA)
	createCA(kluster.Name, "ApiServer Nodes", &certs.ApiServer.Nodes.CA)
	createCA(kluster.Name, "Kubelet Clients", &certs.Kubelet.Clients.CA)
	createCA(kluster.Name, "TLS", &certs.TLS.CA)

	certs.Etcd.Clients.ApiServer = certs.signEtcdClient("apiserver")
	certs.Etcd.Peers.Universal = certs.signEtcdPeer("universal")
	certs.ApiServer.Clients.ClusterAdmin = certs.signApiServerClient("cluster-admin", "system:masters")
	certs.ApiServer.Clients.ControllerManager = certs.signApiServerClient("system:kube-controller-manager")
	certs.ApiServer.Clients.Proxy = certs.signApiServerClient("system:kube-proxy")
	certs.ApiServer.Clients.Scheduler = certs.signApiServerClient("system:kube-scheduler")
	certs.ApiServer.Clients.Wormhole = certs.signApiServerClient("kubernikus:wormhole")
	certs.ApiServer.Nodes.Universal = certs.signApiServerNode("universal")
	certs.Kubelet.Clients.ApiServer = certs.signKubeletClient("apiserver")
	certs.TLS.ApiServer = certs.signTLS("apiserver",
		[]string{"kubernetes", "kubernetes.default", "apiserver", kluster.Name, fmt.Sprintf("%v.%v", kluster.Name, domain)},
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(198, 18, 128, 1)}) // TODO: Make ServiceCidr Configurable
	certs.TLS.Wormhole = certs.signTLS("wormhole",
		[]string{fmt.Sprintf("%v-wormhole.%v", kluster.Name, domain)}, []net.IP{})

	kluster.Secret.Certificates = certs.toMap()
}

func (c Certificates) signEtcdClient(name string) Bundle {
	config := Config{
		Sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.Etcd.Clients.CA.Sign(config)
}

func (c Certificates) signEtcdPeer(name string) Bundle {
	config := Config{
		Sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	return c.Etcd.Peers.CA.Sign(config)
}

func (c Certificates) signApiServerClient(name string, groups ...string) Bundle {
	config := Config{
		Sign:         name,
		Organization: groups,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.ApiServer.Clients.CA.Sign(config)
}

func (c Certificates) signApiServerNode(name string) Bundle {
	config := Config{
		Sign:         name,
		Organization: []string{"system:nodes"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.ApiServer.Nodes.CA.Sign(config)
}

func (c Certificates) signKubeletClient(name string) Bundle {
	config := Config{
		Sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	return c.Kubelet.Clients.CA.Sign(config)
}

func (c Certificates) signTLS(name string, dnsNames []string, ips []net.IP) Bundle {
	config := Config{
		Sign:   name,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames: AltNames{
			DNSNames: dnsNames,
			IPs:      ips,
		},
	}
	return c.TLS.CA.Sign(config)
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
		NotAfter:              now.Add(caValidity).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}

	certDERBytes, _ := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, bundle.PrivateKey.Public(), bundle.PrivateKey)
	bundle.Certificate, _ = x509.ParseCertificate(certDERBytes)
}

func (ca Bundle) Sign(config Config) Bundle {
	if !ca.Certificate.IsCA {
		panic("You can't use this certificate for signing. It's not a CA...")
	}

	if config.ValidFor == 0 {
		config.ValidFor = defaultCertValidity
	}

	key, _ := certutil.NewPrivateKey()
	serial, _ := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:         config.Sign,
			Organization:       config.Organization,
			OrganizationalUnit: ca.Certificate.Subject.OrganizationalUnit,
		},
		DNSNames:     config.AltNames.DNSNames,
		IPAddresses:  config.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    ca.Certificate.NotBefore,
		NotAfter:     time.Now().Add(config.ValidFor).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  config.Usages,
	}

	certDERBytes, _ := x509.CreateCertificate(rand.Reader, &certTmpl, ca.Certificate, key.Public(), ca.PrivateKey)

	cert, _ := x509.ParseCertificate(certDERBytes)

	return Bundle{cert, key}
}
