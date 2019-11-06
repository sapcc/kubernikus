package util

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPOrDNSName(t *testing.T) {

	var result []IPOrDNSName

	require.NoError(t, json.Unmarshal([]byte(`["example.com", "1.1.1.1"]`), &result))

	assert.Equal(t,
		[]IPOrDNSName{
			{Type: DNSNameType, DNSNameVal: "example.com"},
			{Type: IPType, IPVal: net.ParseIP("1.1.1.1")},
		},
		result)
}
