package generator

import (
	"fmt"
	"strings"

	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

type NameGenerator interface {
	// GenerateName generates a valid name from the base name, adding a random suffix to the
	// the base. If base is valid, the returned name must also be valid. The generator is
	// responsible for knowing the maximum valid name length.
	GenerateName(base string) string

	//Prefix generates a valid static prefix without the random suffix
	Prefix(base string) string
}

// simpleNameGenerator generates random names.
type simpleNameGenerator struct{}

// SimpleNameGenerator is a generator that returns the name plus a random suffix of five alphanumerics
// when a name is requested. The string is guaranteed to not exceed the length of a standard Kubernetes
// name (63 characters)
var SimpleNameGenerator NameGenerator = simpleNameGenerator{}

const (
	// TODO: make this flexible for non-core resources with alternate naming rules.
	MaxNameLength          = 63
	RandomLength           = 5
	MaxGeneratedNameLength = MaxNameLength - RandomLength
)

func (simpleNameGenerator) Prefix(base string) string {
	if len(base) > MaxGeneratedNameLength {
		base = base[:MaxGeneratedNameLength]
	}
	return base
}

func (s simpleNameGenerator) GenerateName(base string) string {
	return strings.ToLower(fmt.Sprintf("%s%s", s.Prefix(base), utilrand.String(RandomLength)))
}
