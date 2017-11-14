package models

import (
	"testing"

	strfmt "github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestNodePoolInfoValidation(t *testing.T) {
	info := NodePoolInfo{
		Name: "test",
	}
	assert.NoError(t, info.Validate(strfmt.Default))
	json, err := info.MarshalBinary()
	assert.NoError(t, err)
	assert.Contains(t, string(json), `"size":0`)

}
