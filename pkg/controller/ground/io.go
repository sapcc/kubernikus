package ground

import (
	"fmt"
	"strings"

	"github.com/kennygrant/sanitize"
)

type ConfigPersister interface {
	WriteConfig(Cluster) error
}

type BasePersister struct{}

func (bp BasePersister) basename(b Bundle) string {
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

func (bp BasePersister) nameForKey(b Bundle) string {
	return fmt.Sprintf("%s-key.pem", bp.basename(b))
}

func (bp BasePersister) nameForCert(b Bundle) string {
	return fmt.Sprintf("%s.pem", bp.basename(b))
}
