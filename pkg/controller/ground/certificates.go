package ground

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"path"
	"strings"
	"time"

	certutil "k8s.io/client-go/util/cert"
)

const (
	DEFAULT_CA_RSA_KEY_SIZE = 1024
	DEFAULT_CA_VALIDITY     = time.Hour * 24 * 365 * 10

	ORGANIZATION = "SAP Converged Cloud"
)

type Subject struct {
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

func WriteCertificateAuthorities(name string) {
	writeCertificateAuthority(".", []string{"Etcd", "Peers"}, name)
	writeCertificateAuthority(".", []string{"Etcd", "Clients"}, name)
	writeCertificateAuthority(".", []string{"Kubernetes", "Clients"}, name)
	writeCertificateAuthority(".", []string{"Kubernetes", "Kubelets"}, name)
	writeCertificateAuthority(".", []string{"TLS"}, name)
}

func writeCertificateAuthority(dir string, system []string, project string) error {
	cert, key, err := newCertificateAuthority(system, project)
	if err != nil {
		return err
	}

	err = certutil.WriteCert(pathForCACert(dir, system), certutil.EncodeCertPEM(cert))
	if err != nil {
		return err
	}

	err = certutil.WriteKey(pathForCAKey(dir, system), certutil.EncodePrivateKeyPEM(key))
	if err != nil {
		return err
	}

	return nil
}

func newCertificateAuthority(system []string, project string) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := newPrivateKey(DEFAULT_CA_RSA_KEY_SIZE)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key [%v]", err)
	}

	subject := Subject{
		CommonName:         fmt.Sprintf("%s CA", project),
		Organization:       []string{ORGANIZATION, "Kubernikus"},
		OrganizationalUnit: system,
	}
	cert, err := newSelfSignedCACert(subject, key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create self-signed certificate [%v]", err)
	}

	return cert, key, nil
}

func newPrivateKey(bits int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, bits)
}

func newSelfSignedCACert(subject Subject, key *rsa.PrivateKey) (*x509.Certificate, error) {
	now := time.Now()

	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:         subject.CommonName,
			Organization:       subject.Organization,
			OrganizationalUnit: subject.OrganizationalUnit,
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(DEFAULT_CA_VALIDITY).UTC(),
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

func pathForCert(dir, name string) string {
	return path.Join(dir, fmt.Sprintf("%s.pem", strings.ToLower(name)))
}

func pathForKey(dir, name string) string {
	return path.Join(dir, fmt.Sprintf("%s-key.pem", strings.ToLower(name)))
}

func pathForCACert(dir string, system []string) string {
	return pathForCert(dir, fmt.Sprintf("%s-ca", strings.ToLower(strings.Join(system, "-"))))
}

func pathForCAKey(dir string, system []string) string {
	return pathForKey(dir, fmt.Sprintf("%s-ca", strings.ToLower(strings.Join(system, "-"))))
}
