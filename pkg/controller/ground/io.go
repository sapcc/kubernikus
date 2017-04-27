package ground

import (
	"fmt"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/kennygrant/sanitize"
	certutil "k8s.io/client-go/util/cert"
)

type ConfigPersister interface {
	WriteConfig(Cluster) error
}

type BasePersister struct{}

type HelmValuePersister struct {
	BasePersister

	result *string
}

type FilePersister struct {
	BasePersister
	BaseDir string
}

type HelmValues struct {
	Certs map[string]string
}

func NewFilePersister(basedir string) *FilePersister {
	p := &FilePersister{}
	p.BaseDir = basedir
	return p
}

func NewHelmValuePersister(result *string) *HelmValuePersister {
	p := &HelmValuePersister{}
	p.result = result
	return p
}

func (fp FilePersister) WriteConfig(cluster Cluster) error {
	for _, bundle := range cluster.Certificates.all() {
		err := fp.writeToFiles(bundle)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fp FilePersister) writeToFiles(b *Bundle) error {
	fmt.Println(fp.pathForCert(b))
	err := certutil.WriteCert(fp.pathForCert(b), certutil.EncodeCertPEM(b.Certificate))
	if err != nil {
		return err
	}

	fmt.Println(fp.pathForKey(b))
	err = certutil.WriteKey(fp.pathForKey(b), certutil.EncodePrivateKeyPEM(b.PrivateKey))
	if err != nil {
		return err
	}

	return nil
}

func (bp BasePersister) basename(b *Bundle) string {
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

func (bp BasePersister) nameForKey(b *Bundle) string {
	return fmt.Sprintf("%s-key.pem", bp.basename(b))
}

func (bp BasePersister) nameForCert(b *Bundle) string {
	return fmt.Sprintf("%s.pem", bp.basename(b))
}

func (fp FilePersister) pathForCert(b *Bundle) string {
	return path.Join(fp.basedir(b), fp.nameForCert(b))
}

func (fp FilePersister) pathForKey(b *Bundle) string {
	return path.Join(fp.basedir(b), fp.nameForKey(b))
}

func (fp FilePersister) basedir(b *Bundle) string {
	return sanitize.BaseName(strings.ToLower(b.Certificate.Subject.OrganizationalUnit[len(b.Certificate.Subject.OrganizationalUnit)-1]))
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
