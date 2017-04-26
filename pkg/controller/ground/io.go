package ground

import (
	"fmt"
	"path"
	"strings"

	"github.com/kennygrant/sanitize"
	certutil "k8s.io/client-go/util/cert"
)

type ConfigPersister interface {
	WriteConfig(Cluster) error
}

type HelmValuePersister struct{}
type SpecPersister struct{}

type FilePersister struct {
	BaseDir string
}

func (fp FilePersister) WriteConfig(cluster Cluster) error {
	for _, bundle := range cluster.Certificates.bundles() {
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

func (fp FilePersister) pathForCert(b *Bundle) string {
	return path.Join(fp.basedir(b), fmt.Sprintf("%s.pem", fp.basename(b)))
}

func (fp FilePersister) pathForKey(b *Bundle) string {
	return path.Join(fp.basedir(b), fmt.Sprintf("%s-key.pem", fp.basename(b)))
}

func (fp FilePersister) basename(b *Bundle) string {
	return sanitize.BaseName(strings.ToLower(strings.Join(b.Certificate.Subject.Province, "-")))
}

func (fp FilePersister) basedir(b *Bundle) string {
	return sanitize.BaseName(strings.ToLower(strings.Join(b.Certificate.Subject.Locality, "-")))
}
