package models

import (
	"testing"

	strfmt "github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestNodePoolValidation(t *testing.T) {
	pool := NodePool{
		AvailabilityZone: "something",
		Name:             "test",
		Flavor:           "nase",
		Size:             0,
	}
	assert.NoError(t, pool.Validate(strfmt.Default))
	json, err := pool.MarshalBinary()
	assert.NoError(t, err)
	assert.Contains(t, string(json), `"size":0`)

	pool = NodePool{
		AvailabilityZone: "something",
		Name:             "test_underscore",
		Flavor:           "nase",
		Size:             0,
	}
	assert.Error(t, pool.Validate(strfmt.Default))
}
