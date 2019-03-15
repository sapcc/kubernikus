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

func TestNodePoolTaintValidation(t *testing.T) {
	cases := []struct {
		Taint string
		Valid bool
	}{
		{"valid=taint:NoSchedule", true},
		{"sap.com/valid=taint:NoSchedule", true},
		{"sap.com/valid=taint:NoExecute", true},
		{"sap.com/valid=taint:NoExecute1", false},
		{"sap.com/invalid=taint:InvalidAction", false},
		{"sap._com/invalid=taint:NoSchedule", false},
		{"sap.com/in&valid=taint:NoSchedule", false},
		{".com/in&valid=taint.:NoSchedule", false},
		{"tolongvalue=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab:NoSchedule", false},
	}

	for _, c := range cases {
		pool := NodePool{
			AvailabilityZone: "something",
			Name:             "test",
			Flavor:           "nase",
			Size:             0,
			Taints:           []string{c.Taint},
		}

		err := pool.Validate(strfmt.Default)
		if c.Valid {
			assert.NoError(t, err, "expected taint %s to be valid", c.Taint)
		} else {
			assert.Error(t, err, "expected taint %s to be invalid", c.Taint)
		}
	}

}

func TestNodePoolLabelValidation(t *testing.T) {
	cases := []struct {
		Label string
		Valid bool
	}{
		{"valid=label", true},
		{"sap.com/val-id=label", true},
		{"sap.com/invalid=label:NoExecute", false},
		{"sap._com/invalid=label", false},
		{"sap.com/in&valid=label", false},
		{"sap.com/.invalid=", false},
		{"tolongvalue=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab", false},
	}

	for _, c := range cases {
		pool := NodePool{
			AvailabilityZone: "something",
			Name:             "test",
			Flavor:           "nase",
			Size:             0,
			Labels:           []string{c.Label},
		}

		err := pool.Validate(strfmt.Default)
		if c.Valid {
			assert.NoError(t, err, "expected label %s to be valid", c.Label)
		} else {
			assert.Error(t, err, "expected label %s to be invalid", c.Label)
		}
	}

}
