package ground

import (
	"fmt"
	"path"
	"strings"

	"github.com/kennygrant/sanitize"
	certutil "k8s.io/client-go/util/cert"
)

type FilePersister struct {
	BaseDir string
}

func NewFilePersister(basedir string) *FilePersister {
	p := &FilePersister{}
	p.BaseDir = basedir
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

func (fp FilePersister) writeToFiles(b Bundle) error {
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

func (fp FilePersister) pathForCert(b Bundle) string {
	return path.Join(fp.basedir(b), b.NameForCert())
}

func (fp FilePersister) pathForKey(b Bundle) string {
	return path.Join(fp.basedir(b), b.NameForKey())
}

func (fp FilePersister) basedir(b Bundle) string {
	return sanitize.BaseName(strings.ToLower(b.Certificate.Subject.OrganizationalUnit[len(b.Certificate.Subject.OrganizationalUnit)-1]))
}
