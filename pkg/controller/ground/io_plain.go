package ground

import (
	"fmt"

	certutil "k8s.io/client-go/util/cert"
)

type PlainPersister struct{}

func NewPlainPersister() *PlainPersister {
	return &PlainPersister{}
}

func (fp PlainPersister) WriteConfig(cluster Cluster) error {
	for _, b := range cluster.Certificates.all() {
		fmt.Println(b.NameForCert())
		fmt.Println(string(certutil.EncodeCertPEM(b.Certificate)))
		fmt.Println(b.NameForKey())
		fmt.Println(string(certutil.EncodePrivateKeyPEM(b.PrivateKey)))
	}

	return nil
}
