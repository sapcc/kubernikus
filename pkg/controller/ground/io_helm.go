package ground

import (
	yaml "gopkg.in/yaml.v2"

	certutil "k8s.io/client-go/util/cert"
)

type HelmValuePersister struct {
	BasePersister

	result *string
}

type HelmValues struct {
	Certs map[string]string
}

func NewHelmValuePersister(result *string) *HelmValuePersister {
	p := &HelmValuePersister{}
	p.result = result
	return p
}

func (hvp HelmValuePersister) WriteConfig(cluster Cluster) error {
	values := HelmValues{
		Certs: map[string]string{},
	}

	for _, bundle := range cluster.Certificates.all() {
		values.Certs[hvp.nameForCert(bundle)] = string(certutil.EncodeCertPEM(bundle.Certificate))
		values.Certs[hvp.nameForKey(bundle)] = string(certutil.EncodePrivateKeyPEM(bundle.PrivateKey))
	}

	result, err := yaml.Marshal(&values)
	if err != nil {
		return err
	}

	*hvp.result = string(result)

	return nil
}
