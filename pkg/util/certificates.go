package util

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
	"reflect"
	"time"

	certutil "k8s.io/client-go/util/cert"

	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

const (
	//by default we generate certs with 2 year validity
	defaultCertValidity = 2 * time.Hour * 24 * 365
	//out CAs are valid for 10 years
	caValidity = 10 * time.Hour * 24 * 365
	// renew cert two weeks before it is expired
	certExpiration = 14 * 24 * time.Hour
)

type Bundle struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

func NewBundle(key, cert []byte) (*Bundle, error) {
	certificates, err := certutil.ParseCertsPEM(cert)
	if err != nil {
		return nil, err
	}
	if len(certificates) < 1 {
		return nil, errors.New("No certificates found")
	}
	k, err := certutil.ParsePrivateKeyPEM(key)
	if err != nil {
		return nil, err
	}
	rsaKey, isRSAKey := k.(*rsa.PrivateKey)
	if !isRSAKey {
		return nil, errors.New("Key does not seem to be of type RSA")
	}

	return &Bundle{PrivateKey: rsaKey, Certificate: certificates[0]}, nil
}

type Config struct {
	Sign               string
	Organization       []string
	OrganizationalUnit []string
	Province           []string
	Locality           []string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
	ValidFor           time.Duration
}

type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

func (ca Bundle) Sign(config Config) (*Bundle, error) {
	if !ca.Certificate.IsCA {
		return nil, errors.New("You can't use this certificate for signing. It's not a CA...")
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
			OrganizationalUnit: config.OrganizationalUnit,
			Province:           config.Province,
			Locality:           config.Locality,
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

	return &Bundle{cert, key}, nil
}

type CertificateFactory struct {
	kluster     *v1.Kluster
	store       *v1.Certificates
	domain      string
	certUpdates []CertUpdates
}

type CertUpdates struct {
	Certificate *x509.Certificate
	Type        string
	Reason      string
}

func NewCertificateFactory(kluster *v1.Kluster, store *v1.Certificates, domain string) *CertificateFactory {
	return &CertificateFactory{kluster, store, domain, []CertUpdates{}}
}

func (cf *CertificateFactory) Ensure() error {
	apiServiceIP, err := cf.kluster.ApiServiceIP()
	if err != nil {
		return err
	}

	apiIP := net.ParseIP(cf.kluster.Spec.AdvertiseAddress)
	if apiIP == nil {
		return fmt.Errorf("Failed to parse clusters advertise address: %s", cf.kluster.Spec.AdvertiseAddress)
	}

	cf.certUpdates = []CertUpdates{}

	etcdClientsCA, err := loadOrCreateCA(cf.kluster, "Etcd Clients", &cf.store.EtcdClientsCACertificate, &cf.store.EtcdClientsCAPrivateKey)
	if err != nil {
		return err
	}
	_, err = loadOrCreateCA(cf.kluster, "Etcd Peers", &cf.store.EtcdPeersCACertificate, &cf.store.EtcdPeersCAPrivateKey)
	if err != nil {
		return err
	}
	apiserverClientsCA, err := loadOrCreateCA(cf.kluster, "ApiServer Clients", &cf.store.ApiserverClientsCACertifcate, &cf.store.ApiserverClientsCAPrivateKey)
	if err != nil {
		return err
	}
	_, err = loadOrCreateCA(cf.kluster, "ApiServer Nodes", &cf.store.ApiserverNodesCACertificate, &cf.store.ApiserverNodesCAPrivateKey)
	if err != nil {
		return err
	}
	kubeletClientsCA, err := loadOrCreateCA(cf.kluster, "Kubelet Clients", &cf.store.KubeletClientsCACertificate, &cf.store.KubeletClientsCAPrivateKey)
	if err != nil {
		return err
	}
	tlsCA, err := loadOrCreateCA(cf.kluster, "TLS", &cf.store.TLSCACertificate, &cf.store.TLSCAPrivateKey)
	if err != nil {
		return err
	}
	aggregationCA, err := loadOrCreateCA(cf.kluster, "Aggregation", &cf.store.AggregationCACertificate, &cf.store.AggregationCAPrivateKey)
	if err != nil {
		return err
	}

	if err := ensureClientCertificate(
		etcdClientsCA,
		"apiserver",
		nil,
		&cf.store.EtcdClientsApiserverCertificate,
		&cf.store.EtcdClientsApiserverPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureClientCertificate(apiserverClientsCA,
		"cluster-admin",
		[]string{"system:masters"},
		&cf.store.ApiserverClientsClusterAdminCertificate,
		&cf.store.ApiserverClientsClusterAdminPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureClientCertificate(apiserverClientsCA,
		"system:kube-controller-manager",
		nil,
		&cf.store.ApiserverClientsKubeControllerManagerCertificate,
		&cf.store.ApiserverClientsKubeControllerManagerPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureClientCertificate(
		apiserverClientsCA,
		"system:kube-proxy",
		nil,
		&cf.store.ApiserverClientsKubeProxyCertificate,
		&cf.store.ApiserverClientsKubeProxyPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureClientCertificate(
		apiserverClientsCA,
		"system:kube-scheduler",
		nil,
		&cf.store.ApiserverClientsKubeSchedulerCertificate,
		&cf.store.ApiserverClientsKubeSchedulerPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureClientCertificate(
		apiserverClientsCA,
		"kubernikus:wormhole",
		nil,
		&cf.store.ApiserverClientsKubernikusWormholeCertificate,
		&cf.store.ApiserverClientsKubernikusWormholePrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureClientCertificate(
		kubeletClientsCA,
		"apiserver",
		nil,
		&cf.store.KubeletClientsApiserverCertificate,
		&cf.store.KubeletClientsApiserverPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureClientCertificate(
		aggregationCA,
		"aggregator",
		nil,
		&cf.store.AggregationAggregatorCertificate,
		&cf.store.AggregationAggregatorPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}

	if err := ensureServerCertificate(tlsCA, "apiserver",
		[]string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "apiserver", cf.kluster.Name, fmt.Sprintf("%s.%s", cf.kluster.Name, cf.kluster.Namespace), fmt.Sprintf("%v.%v", cf.kluster.Name, cf.domain)},
		[]net.IP{net.IPv4(127, 0, 0, 1), apiServiceIP, apiIP},
		&cf.store.TLSApiserverCertificate,
		&cf.store.TLSApiserverPrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}
	if err := ensureServerCertificate(tlsCA, "wormhole",
		[]string{fmt.Sprintf("%v-wormhole.%v", cf.kluster.Name, cf.domain)},
		nil,
		&cf.store.TLSWormholeCertificate,
		&cf.store.TLSWormholePrivateKey,
		&cf.certUpdates); err != nil {
		return err
	}

	return nil
}

func (cf *CertificateFactory) UserCert(principal *models.Principal, apiURL string) (*Bundle, error) {

	caBundle, err := NewBundle([]byte(cf.store.ApiserverClientsCAPrivateKey), []byte(cf.store.ApiserverClientsCACertifcate))
	if err != nil {
		return nil, err
	}

	var organizations []string
	for _, role := range principal.Roles {
		organizations = append(organizations, "os:"+role)
	}
	projectid := cf.kluster.Spec.Openstack.ProjectID
	//nocloud clusters don't have the projectID in the spec
	if projectid == "" {
		projectid = cf.kluster.Account()
	}

	return caBundle.Sign(Config{
		Sign:         fmt.Sprintf("%s@%s", principal.Name, principal.Domain),
		Organization: organizations,
		Province:     []string{principal.AuthURL, projectid},
		Locality:     []string{apiURL},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ValidFor:     24 * time.Hour,
	})

}

func (cf *CertificateFactory) GetCertUpdates() []CertUpdates {
	return cf.certUpdates
}

func loadOrCreateCA(kluster *v1.Kluster, name string, cert, key *string) (*Bundle, error) {
	if *cert != "" && *key != "" {
		return NewBundle([]byte(*key), []byte(*cert))
	}
	caBundle, err := createCA(kluster.Name, name)
	if err != nil {
		return nil, err
	}
	*cert = string(certutil.EncodeCertPEM(caBundle.Certificate))
	*key = string(certutil.EncodePrivateKeyPEM(caBundle.PrivateKey))
	return caBundle, nil
}

func ensureClientCertificate(ca *Bundle, cn string, groups []string, cert, key *string, certUpdates *[]CertUpdates) error {
	certificate, err := ca.Sign(Config{
		Sign:         cn,
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Organization: groups,
	})
	if err != nil {
		return err
	}

	if *cert != "" && *key != "" {
		certBundle, err := NewBundle([]byte(*key), []byte(*cert))

		if err != nil {
			return fmt.Errorf("Failed parsing certificate bundle: %s", err)
		}

		reason, invalid := isCertChangedOrExpires(certBundle.Certificate, certificate.Certificate, certExpiration)
		if !invalid {
			return nil
		}

		update := CertUpdates{
			Certificate: certBundle.Certificate,
			Type:        "Client Certificate",
			Reason:      reason,
		}
		*certUpdates = append(*certUpdates, update)
	}

	*cert = string(certutil.EncodeCertPEM(certificate.Certificate))
	*key = string(certutil.EncodePrivateKeyPEM(certificate.PrivateKey))
	return nil

}

func ensureServerCertificate(ca *Bundle, cn string, dnsNames []string, ips []net.IP, cert, key *string, certUpdates *[]CertUpdates) error {
	certificate, err := ca.Sign(Config{
		Sign:   cn,
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames: AltNames{
			DNSNames: dnsNames,
			IPs:      ips,
		},
	})
	if err != nil {
		return err
	}

	if *cert != "" && *key != "" {
		certBundle, err := NewBundle([]byte(*key), []byte(*cert))

		if err != nil {
			return fmt.Errorf("Failed parsing certificate bundle: %s", err)
		}

		reason, invalid := isCertChangedOrExpires(certBundle.Certificate, certificate.Certificate, certExpiration)
		if !invalid {
			return nil
		}

		update := CertUpdates{
			Certificate: certBundle.Certificate,
			Type:        "Server Certificate",
			Reason:      reason,
		}
		*certUpdates = append(*certUpdates, update)
	}

	*cert = string(certutil.EncodeCertPEM(certificate.Certificate))
	*key = string(certutil.EncodePrivateKeyPEM(certificate.PrivateKey))
	return nil
}

func createCA(klusterName, name string) (*Bundle, error) {
	privateKey, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate private key for %s ca: %s", name, err)
	}

	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:         name,
			OrganizationalUnit: []string{CA_ISSUER_KUBERNIKUS_IDENTIFIER_0, CA_ISSUER_KUBERNIKUS_IDENTIFIER_1, klusterName},
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(caValidity).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, privateKey.Public(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to create certificate for %s CA: %s", name, err)
	}
	certificate, err := x509.ParseCertificate(certDERBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse cert for %s CA: %s", name, err)
	}
	return &Bundle{PrivateKey: privateKey, Certificate: certificate}, nil
}

func isCertChangedOrExpires(origCert, newCert *x509.Certificate, duration time.Duration) (string, bool) {
	if !reflect.DeepEqual(origCert.DNSNames, newCert.DNSNames) {
		return "DNS changes", true
	}

	if !reflect.DeepEqual(origCert.IPAddresses, newCert.IPAddresses) {
		return "IP changes", true
	}

	expire := time.Now().Add(duration)
	if expire.After(origCert.NotAfter) {
		return fmt.Sprintf("Certificate expires at %s", origCert.NotAfter), true
	}

	return "", false
}
