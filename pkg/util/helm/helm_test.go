package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func TestKlusterToHelmValues(t *testing.T) {
	kluster := &v1.Kluster{}
	secret := &v1.Secret{}
	secret.ExtraValues = "a: a\n" +
		"etcd:\n" +
		"  backup: b\n"
	b, err := KlusterToHelmValues(kluster, secret, "v1.10.4", nil, "doof")
	// There should be no error converting this
	assert.NoError(t, err)
	// Make sure easy stuff is in
	assert.Equal(t, b["a"], "a")
	bck := b["etcd"]
	// The etcd entry should stay an object
	assert.IsType(t, map[string]interface{}{}, bck)
	mbck := bck.(map[string]interface{})
	// Make sure the object was overwritten
	assert.Equal(t, mbck["backup"], "b")
}
