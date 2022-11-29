package util

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"reflect"
	"strings"
	"time"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"

	"github.com/sapcc/kubernikus/pkg/api/auth"
	"github.com/sapcc/kubernikus/pkg/api/models"
	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

const (
	//by default we generate certs with 2 year validity
	defaultCertValidity = 2 * time.Hour * 24 * 365
	//out CAs are valid for 10 years
	caValidity = 10 * time.Hour * 24 * 365
	// renew certs 90 days before they expire
	certExpiration                    = 90 * 24 * time.Hour
	AdditionalApiserverSANsAnnotation = "kubernikus.cloud.sap/additional-apiserver-sans"
	AdditionalWormholeSANsAnnotation  = "kubernikus.cloud.sap/additional-wormhole-sans"
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
	k, err := keyutil.ParsePrivateKeyPEM(key)
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

	key, _ := NewPrivateKey()
	serial, _ := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))

	//backdate not before to compensate clock skew
	notBefore := time.Now().Add(-1 * time.Hour)
	//Don't create certs that are valid before the issuing CA
	if ca.Certificate.NotBefore.After(notBefore) {
		notBefore = ca.Certificate.NotBefore
	}

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
		NotBefore:    notBefore,
		NotAfter:     time.Now().Add(config.ValidFor).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  config.Usages,
	}

	certDERBytes, _ := x509.CreateCertificate(cryptorand.Reader, &certTmpl, ca.Certificate, key.Public(), ca.PrivateKey)

	cert, _ := x509.ParseCertificate(certDERBytes)

	return &Bundle{cert, key}, nil
}

type CertificateFactory struct {
	kluster *v1.Kluster
	store   *v1.Certificates
	domain  string
}

type CertUpdates struct {
	Type   string
	Name   string
	Reason string
}

func NewCertificateFactory(kluster *v1.Kluster, store *v1.Certificates, domain string) *CertificateFactory {
	return &CertificateFactory{kluster, store, domain}
}

func (cf *CertificateFactory) Ensure() ([]CertUpdates, error) {
	apiServiceIP, err := cf.kluster.ApiServiceIP()
	if err != nil {
		return nil, err
	}

	apiIP := net.ParseIP(cf.kluster.Spec.AdvertiseAddress)
	if apiIP == nil {
		return nil, fmt.Errorf("Failed to parse clusters advertise address: %s", cf.kluster.Spec.AdvertiseAddress)
	}

	certUpdates := []CertUpdates{}

	tlsEtcdCA, err := loadOrCreateCA(cf.kluster, "TLSEtcd", &cf.store.TLSEtcdCACertificate, &cf.store.TLSEtcdCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}
	etcdClientsCA, err := loadOrCreateCA(cf.kluster, "Etcd Clients", &cf.store.EtcdClientsCACertificate, &cf.store.EtcdClientsCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}
	_, err = loadOrCreateCA(cf.kluster, "Etcd Peers", &cf.store.EtcdPeersCACertificate, &cf.store.EtcdPeersCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}
	apiserverClientsCA, err := loadOrCreateCA(cf.kluster, "ApiServer Clients", &cf.store.ApiserverClientsCACertifcate, &cf.store.ApiserverClientsCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}
	_, err = loadOrCreateCA(cf.kluster, "ApiServer Nodes", &cf.store.ApiserverNodesCACertificate, &cf.store.ApiserverNodesCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}
	kubeletClientsCA, err := loadOrCreateCA(cf.kluster, "Kubelet Clients", &cf.store.KubeletClientsCACertificate, &cf.store.KubeletClientsCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}
	tlsCA, err := loadOrCreateCA(cf.kluster, "TLS", &cf.store.TLSCACertificate, &cf.store.TLSCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}
	aggregationCA, err := loadOrCreateCA(cf.kluster, "Aggregation", &cf.store.AggregationCACertificate, &cf.store.AggregationCAPrivateKey, &certUpdates)
	if err != nil {
		return nil, err
	}

	if err := ensureClientCertificate(
		etcdClientsCA,
		"apiserver",
		nil,
		&cf.store.EtcdClientsApiserverCertificate,
		&cf.store.EtcdClientsApiserverPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(
		etcdClientsCA,
		"dex",
		nil,
		&cf.store.EtcdClientsDexCertificate,
		&cf.store.EtcdClientsDexPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(apiserverClientsCA,
		"cluster-admin",
		[]string{"system:masters"},
		&cf.store.ApiserverClientsClusterAdminCertificate,
		&cf.store.ApiserverClientsClusterAdminPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(etcdClientsCA,
		"backup",
		nil,
		&cf.store.EtcdClientsBackupCertificate,
		&cf.store.EtcdClientsBackupPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(apiserverClientsCA,
		"system:kube-controller-manager",
		nil,
		&cf.store.ApiserverClientsKubeControllerManagerCertificate,
		&cf.store.ApiserverClientsKubeControllerManagerPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(
		apiserverClientsCA,
		"system:kube-proxy",
		nil,
		&cf.store.ApiserverClientsKubeProxyCertificate,
		&cf.store.ApiserverClientsKubeProxyPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(
		apiserverClientsCA,
		"system:kube-scheduler",
		nil,
		&cf.store.ApiserverClientsKubeSchedulerCertificate,
		&cf.store.ApiserverClientsKubeSchedulerPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(
		apiserverClientsCA,
		"kubernikus:wormhole",
		nil,
		&cf.store.ApiserverClientsKubernikusWormholeCertificate,
		&cf.store.ApiserverClientsKubernikusWormholePrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(
		apiserverClientsCA,
		"system:serviceaccount:kube-system:csi-cinder-controller-sa",
		nil,
		&cf.store.ApiserverClientsCSIControllerCertificate,
		&cf.store.ApiserverClientsCSIControllerPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(
		kubeletClientsCA,
		"apiserver",
		nil,
		&cf.store.KubeletClientsApiserverCertificate,
		&cf.store.KubeletClientsApiserverPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureClientCertificate(
		aggregationCA,
		"aggregator",
		nil,
		&cf.store.AggregationAggregatorCertificate,
		&cf.store.AggregationAggregatorPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}

	apiServerDNSNames := []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "apiserver", cf.kluster.Name, fmt.Sprintf("%s.%s", cf.kluster.Name, cf.kluster.Namespace), fmt.Sprintf("%v.%v", cf.kluster.Name, cf.domain)}
	apiServerIPs := []net.IP{net.IPv4(127, 0, 0, 1), apiServiceIP, apiIP}
	if ann := cf.kluster.Annotations[AdditionalApiserverSANsAnnotation]; ann != "" {
		dnsNames, ips, err := addtionalSANsFromAnnotation(ann)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse annotation %s: %v", AdditionalApiserverSANsAnnotation, err)
		}
		apiServerDNSNames = append(apiServerDNSNames, dnsNames...)
		apiServerIPs = append(apiServerIPs, ips...)
	}
	if err := ensureServerCertificate(tlsCA, "apiserver",
		apiServerDNSNames,
		apiServerIPs,
		&cf.store.TLSApiserverCertificate,
		&cf.store.TLSApiserverPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}

	wormholeDNSNames := []string{fmt.Sprintf("%v-wormhole.%v", cf.kluster.Name, cf.domain), "apiserver-proxy-pod-webhook.kube-system.svc"} // hack for apiserver-proxy-pod-webhook
	if ann := cf.kluster.Annotations[AdditionalWormholeSANsAnnotation]; ann != "" {
		dnsNames, _, err := addtionalSANsFromAnnotation(ann)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse annotation %s: %v", AdditionalWormholeSANsAnnotation, err)
		}
		wormholeDNSNames = append(wormholeDNSNames, dnsNames...)
	}
	if err := ensureServerCertificate(tlsCA, "wormhole",
		wormholeDNSNames,
		[]net.IP{net.IPv4(147, 204, 33, 50)}, // hack for konnectivity lb
		&cf.store.TLSWormholeCertificate,
		&cf.store.TLSWormholePrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}
	if err := ensureServerCertificate(tlsEtcdCA, "etcd",
		[]string{fmt.Sprintf("%v-etcd", cf.kluster.Name), fmt.Sprintf("%v-etcd.%v", cf.kluster.Name, cf.domain), "localhost"},
		[]net.IP{net.IPv4(127, 0, 0, 1)},
		&cf.store.TLSEtcdCertificate,
		&cf.store.TLSEtcdPrivateKey,
		&certUpdates); err != nil {
		return nil, err
	}

	return certUpdates, nil
}

func (cf *CertificateFactory) UserCert(principal *models.Principal, apiURL string) (*Bundle, error) {

	caBundle, err := NewBundle([]byte(cf.store.ApiserverClientsCAPrivateKey), []byte(cf.store.ApiserverClientsCACertifcate))
	if err != nil {
		return nil, err
	}

	var organizations []string
	for _, group := range principal.Groups {
		organizations = append(organizations, group)
	}
	for _, role := range principal.Roles {
		organizations = append(organizations, "os:"+role)
	}
	projectid := cf.kluster.Account()
	cn := principal.Name
	if principal.Domain != "" {
		cn = fmt.Sprintf("%s@%s", principal.Name, principal.Domain)
	}

	province := []string{projectid}
	if a := auth.OpenStackAuthURL(); a != "" {
		province = append([]string{a}, province...)
	}

	return caBundle.Sign(Config{
		Sign:         cn,
		Organization: organizations,
		Province:     province,
		Locality:     []string{apiURL},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ValidFor:     24 * time.Hour,
	})

}

func loadOrCreateCA(kluster *v1.Kluster, name string, cert, key *string, certUpdates *[]CertUpdates) (*Bundle, error) {
	if *cert != "" && *key != "" {
		return NewBundle([]byte(*key), []byte(*cert))
	}
	caBundle, err := createCA(kluster.Name, name)
	if err != nil {
		return nil, err
	}

	update := CertUpdates{
		Type:   "CA certificate",
		Name:   name,
		Reason: "CA missing",
	}
	*certUpdates = append(*certUpdates, update)

	*cert = string(EncodeCertPEM(caBundle.Certificate))
	*key = string(EncodePrivateKeyPEM(caBundle.PrivateKey))
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

	reason := ""

	if *cert != "" && *key != "" {
		certBundle, err := NewBundle([]byte(*key), []byte(*cert))

		if err != nil {
			return fmt.Errorf("Failed parsing certificate bundle: %s", err)
		}

		var invalid bool
		reason, invalid = isCertChangedOrExpires(certBundle.Certificate, certificate.Certificate, ca.Certificate, certExpiration)
		if !invalid {
			return nil
		}
	}

	if reason == "" {
		reason = "Client certificate missing"
	}

	update := CertUpdates{
		Type:   "Client Certificate",
		Name:   cn,
		Reason: reason,
	}
	*certUpdates = append(*certUpdates, update)

	*cert = string(EncodeCertPEM(certificate.Certificate))
	*key = string(EncodePrivateKeyPEM(certificate.PrivateKey))
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

	reason := ""

	if *cert != "" && *key != "" {
		certBundle, err := NewBundle([]byte(*key), []byte(*cert))

		if err != nil {
			return fmt.Errorf("Failed parsing certificate bundle: %s", err)
		}

		var invalid bool
		reason, invalid = isCertChangedOrExpires(certBundle.Certificate, certificate.Certificate, ca.Certificate, certExpiration)
		if !invalid {
			return nil
		}
	}

	if reason == "" {
		reason = "Server certificate missing"
	}

	update := CertUpdates{
		Type:   "Server Certificate",
		Name:   cn,
		Reason: reason,
	}
	*certUpdates = append(*certUpdates, update)

	*cert = string(EncodeCertPEM(certificate.Certificate))
	*key = string(EncodePrivateKeyPEM(certificate.PrivateKey))
	return nil
}

func createCA(klusterName, name string) (*Bundle, error) {
	privateKey, err := NewPrivateKey()
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

	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &tmpl, &tmpl, privateKey.Public(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to create certificate for %s CA: %s", name, err)
	}
	certificate, err := x509.ParseCertificate(certDERBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse cert for %s CA: %s", name, err)
	}
	return &Bundle{PrivateKey: privateKey, Certificate: certificate}, nil
}

func isCertChangedOrExpires(origCert, newCert, caCert *x509.Certificate, duration time.Duration) (string, bool) {
	if !reflect.DeepEqual(origCert.DNSNames, newCert.DNSNames) {
		return "SAN DNS changes: " + strings.Join(StringSliceDiff(origCert.DNSNames, newCert.DNSNames), " "), true
	}

	if !reflect.DeepEqual(origCert.IPAddresses, newCert.IPAddresses) {
		return "SAN IP changes: " + strings.Join(IPSliceDiff(origCert.IPAddresses, newCert.IPAddresses), " "), true
	}

	expire := time.Now().Add(duration)
	if expire.After(origCert.NotAfter) {
		return fmt.Sprintf("Certificate expires at %s", origCert.NotAfter), true
	}

	err := origCert.CheckSignatureFrom(caCert)
	if err != nil {
		return fmt.Sprintf("CA certificate signature change: %s", err), true
	}

	return "", false
}

func StringSliceDiff(o, n []string) []string {
	oInt := make([]interface{}, len(o), len(o))
	for i := range o {
		oInt[i] = o[i]
	}
	nInt := make([]interface{}, len(n), len(n))
	for i := range n {
		nInt[i] = n[i]
	}
	return SliceDiff(oInt, nInt)
}

func IPSliceDiff(o, n []net.IP) []string {
	oInt := make([]interface{}, len(o), len(o))
	for i := range o {
		oInt[i] = o[i]
	}
	nInt := make([]interface{}, len(n), len(n))
	for i := range n {
		nInt[i] = n[i]
	}
	return SliceDiff(oInt, nInt)
}

func SliceDiff(oldSlice, newSlice []interface{}) []string {
	diff := []string{}
	//addtions
OUTER:
	for _, n := range newSlice {
		for _, o := range oldSlice {
			if reflect.DeepEqual(n, o) {
				continue OUTER
			}
		}
		diff = append(diff, fmt.Sprintf("+%v", n))
	}
OUTER2:
	for _, o := range oldSlice {
		for _, n := range newSlice {
			if reflect.DeepEqual(o, n) {
				continue OUTER2
			}
		}
		diff = append(diff, fmt.Sprintf("-%v", o))
	}
	return diff
}

func addtionalSANsFromAnnotation(ann string) (dnsNames []string, ips []net.IP, err error) {
	var additionalValues []IPOrDNSName
	if err = json.Unmarshal([]byte(ann), &additionalValues); err != nil {
		return
	}
	for _, ipOrName := range additionalValues {
		switch ipOrName.Type {
		case IPType:
			ips = append(ips, ipOrName.IPVal)
		case DNSNameType:
			dnsNames = append(dnsNames, ipOrName.DNSNameVal)
		default:
			return nil, nil, fmt.Errorf("impossible IPOrDNSName.Type")
		}
	}
	return
}

const (
	// PrivateKeyBlockType is a possible value for pem.Block.Type.
	PrivateKeyBlockType = "PRIVATE KEY"
	// PublicKeyBlockType is a possible value for pem.Block.Type.
	PublicKeyBlockType = "PUBLIC KEY"
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
	// RSAPrivateKeyBlockType is a possible value for pem.Block.Type.
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
	rsaKeySize             = 2048
)

func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// EncodePrivateKeyPEM returns PEM-encoded private key data
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  RSAPrivateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}

// NewPrivateKey creates an RSA private key
func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}
