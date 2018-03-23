package ip

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIDROverlap(t *testing.T) {

	testCases := []struct {
		cidr1  string
		cidr2  string
		output bool
	}{
		{"10.0.0.0/8", "192.168.0.0/24", false},
		{"10.0.0.0/8", "10.0.0.0/8", true},
		{"10.0.0.0/8", "10.0.1.0/24", true},
		{"10.0.1.0/24", "10.0.0.0/8", true},
	}
	for n, c := range testCases {
		_, cidr1, _ := net.ParseCIDR(c.cidr1)
		_, cidr2, _ := net.ParseCIDR(c.cidr2)
		result := CIDROverlap(cidr1, cidr2)
		assert.Equal(t, c.output, result, "case number %d failed", n+1)
	}

}
