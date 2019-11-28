package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func TestKlusterToHelmValues(t *testing.T) {
	kluster := &v1.Kluster{}
	secret := &v1.Secret{}
	secret.ExtraValues = make(map[interface{}]interface{})
	secret.ExtraValues["a"] = "a"
	secret.ExtraValues["etcd"] = map[interface{}]interface{}{"backup": "b"}
	b, err := KlusterToHelmValues(kluster, secret, "v1.10.4", nil, "doof")
	// There should be no error converting this
	assert.NoError(t, err)
	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal(b, m)
	// There shold be no error reconverting this
	assert.NoError(t, err)
	// Make sure easy stuff is in
	assert.Equal(t, m["a"], "a")
	bck := m["etcd"]
	// The etcd entry should stay an object
	assert.IsType(t, map[interface{}]interface{}{}, bck)
	mbck := bck.(map[interface{}]interface{})
	// Make sure the object was overwritten
	assert.Equal(t, mbck["backup"], "b")
}
