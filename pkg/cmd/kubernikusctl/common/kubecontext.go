package common

import (
	"crypto/x509"
	"encoding/pem"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/sapcc/kubernikus/pkg/util"
)

func NewKubernikusContext(kubeconfig, context string) (*KubernikusContext, error) {

	pathOptions := clientcmd.NewDefaultPathOptions()
	pathOptions.LoadingRules.ExplicitPath = kubeconfig

	config, err := pathOptions.GetStartingConfig()
	if err != nil {
		return nil, err
	}

	ktx := &KubernikusContext{Config: config, PathOptions: pathOptions, context: context}

	return ktx, err
}

type KubernikusContext struct {
	PathOptions *clientcmd.PathOptions
	Config      *clientcmdapi.Config
	context     string
	caCert      *x509.Certificate
	clientCert  *x509.Certificate
}

func (ktx *KubernikusContext) IsKubernikusContext() (bool, error) {
	caCert, err := ktx.getCACertificate()
	if err != nil {
		return false, err
	}

	//With go 1.16, the order of multivalued fields in parsed certs became unreliable: https://github.com/golang/go/issues/45882
	return sets.NewString(caCert.Subject.OrganizationalUnit...).HasAll(util.CA_ISSUER_KUBERNIKUS_IDENTIFIER_0, util.CA_ISSUER_KUBERNIKUS_IDENTIFIER_1), nil

}

func (ktx *KubernikusContext) UserCertificateValid() (bool, error) {
	cert, err := ktx.getClientCertificate()
	if err != nil {
		return false, err
	}

	return time.Now().After(cert.NotBefore) && time.Now().Before(cert.NotAfter), nil
}

func (ktx *KubernikusContext) Username() (string, error) {
	cert, err := ktx.getClientCertificate()
	if err != nil {
		return "", err
	}

	parts := strings.Split(cert.Subject.CommonName, "@")
	if len(parts) != 2 {
		return "", errors.Errorf("Couldn't extract username/domain from client certificate %v", parts)
	}
	return parts[0], nil
}

func (ktx *KubernikusContext) UserDomainname() (string, error) {
	cert, err := ktx.getClientCertificate()
	if err != nil {
		return "", err
	}

	parts := strings.Split(cert.Subject.CommonName, "@")
	if len(parts) != 2 {
		return "", errors.Errorf("Couldn't extract username/domain from client certificate %v", parts)
	}
	return parts[1], nil
}

func (ktx *KubernikusContext) KubernikusURL() (string, error) {
	cert, err := ktx.getClientCertificate()
	if err != nil {
		return "", err
	}

	if len(cert.Subject.Locality) == 0 {
		return "", errors.Errorf("CA certificate didn't contain Kubernikus metadata")
	}
	return cert.Subject.Locality[0], nil
}

func (ktx *KubernikusContext) AuthURL() (string, error) {
	cert, err := ktx.getClientCertificate()
	if err != nil {
		return "", err
	}

	//With go 1.16, the order of multivalued fields in parsed certs became unreliable: https://github.com/golang/go/issues/45882
	for _, p := range cert.Subject.Province {
		if strings.HasPrefix(p, "http") {
			return p, nil
		}
	}
	return "", errors.Errorf("Client certificate didn't contain OpenStack metadata")
}

func (ktx *KubernikusContext) ProjectID() (string, error) {
	cert, err := ktx.getClientCertificate()
	if err != nil {
		return "", err
	}
	if len(cert.Subject.Province) < 2 {
		return "", errors.New("Client certificate is missing kubernikus metadata")
	}
	//With go 1.16, the order of multivalued fields in parsed certs became unreliable: https://github.com/golang/go/issues/45882
	for _, p := range cert.Subject.Province {
		if !strings.HasPrefix(p, "http") {
			return p, nil
		}
	}
	return "", errors.New("Client certificate didn't contain OpenStack metadata")
}

func (ktx *KubernikusContext) MergeAndPersist(rawConfig string) error {
	config, err := clientcmd.Load([]byte(rawConfig))
	if err != nil {
		return errors.Wrapf(err, "Couldn't load kubernikus kubeconfig: %v", rawConfig)
	}

	//don't change the current context if there is already one set
	if ktx.Config.CurrentContext != "" {
		config.CurrentContext = ""
	}
	if err := mergo.MergeWithOverwrite(ktx.Config, config); err != nil {
		return errors.Wrap(err, "Couldn't merge kubeconfigs")
	}

	if err = clientcmd.ModifyConfig(ktx.PathOptions, *ktx.Config, false); err != nil {
		return errors.Wrapf(err, "Couldn't merge Kubernikus config with kubeconfig at %v:", ktx.PathOptions.GetDefaultFilename())
	}

	return nil
}

func (ktx *KubernikusContext) getCACertificate() (*x509.Certificate, error) {
	if ktx.caCert != nil {
		return ktx.caCert, nil
	}
	data, err := ktx.getRawCACertificate()
	if err != nil {
		return nil, err
	}
	c, err := parseRawPEM(data)
	if err != nil {
		return nil, err
	}
	ktx.caCert = c
	return ktx.caCert, nil
}

func (ktx *KubernikusContext) getRawCACertificate() ([]byte, error) {
	context := ktx.Config.Contexts[ktx.context]
	if context == nil {
		return nil, errors.Errorf("Couldn't find context %v", ktx.context)
	}

	authInfo := ktx.Config.AuthInfos[context.AuthInfo]
	if authInfo == nil {
		return nil, errors.Errorf("Couldn't find auth-info %v for context %v", context.AuthInfo, ktx.context)
	}

	cluster := ktx.Config.Clusters[context.Cluster]
	if cluster == nil {
		return nil, errors.Errorf("Couldn't find cluster %v", context.Cluster)
	}

	certData := cluster.CertificateAuthorityData
	if certData == nil {
		return nil, errors.Errorf("Couldn't find CA certificate for cluster %v", context.Cluster)
	}

	return certData, nil
}

func (ktx *KubernikusContext) getClientCertificate() (*x509.Certificate, error) {
	if ktx.clientCert != nil {
		return ktx.clientCert, nil
	}
	data, err := ktx.getRawClientCertificate()
	if err != nil {
		return nil, err
	}
	c, err := parseRawPEM(data)
	if err != nil {
		return nil, err
	}
	ktx.clientCert = c
	return ktx.clientCert, nil
}

func (ktx *KubernikusContext) getRawClientCertificate() ([]byte, error) {
	context := ktx.Config.Contexts[ktx.context]
	if context == nil {
		return nil, errors.Errorf("Couldn't find context %v", ktx.context)
	}

	authInfo := ktx.Config.AuthInfos[context.AuthInfo]
	if authInfo == nil {
		return nil, errors.Errorf("Couldn't find auth-info %v for context %v", context.AuthInfo, ktx.context)
	}

	cluster := ktx.Config.Clusters[context.Cluster]
	if cluster == nil {
		return nil, errors.Errorf("Couldn't find cluster %v", context.Cluster)
	}

	certData := authInfo.ClientCertificateData
	if certData == nil {
		return nil, errors.Errorf("Couldn't find client certificate for auth-info %v", authInfo.Username)
	}

	return certData, nil
}

func parseRawPEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("Couldn't decode raw certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't parse certificate")
	}

	return cert, nil
}
