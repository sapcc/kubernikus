package util

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceDiff(t *testing.T) {

	cases := []struct {
		Old    []interface{}
		New    []interface{}
		Result []string
	}{
		{
			Old:    []interface{}{"a", "b", "c"},
			New:    []interface{}{"b", "c", "d"},
			Result: []string{"+d", "-a"},
		},
		{
			Old:    []interface{}{"a", net.IPv4(1, 1, 1, 1)},
			New:    []interface{}{"b", net.IPv4(2, 2, 2, 2)},
			Result: []string{"+b", "+2.2.2.2", "-a", "-1.1.1.1"},
		},
	}

	for n, c := range cases {
		assert.Equal(t, c.Result, SliceDiff(c.Old, c.New), "Test case %d failed", n)
	}

	assert.Equal(t, []string{"+2.2.2.2", "-1.1.1.1"}, IPSliceDiff([]net.IP{net.IPv4(1, 1, 1, 1)}, []net.IP{net.IPv4(2, 2, 2, 2)}))
	assert.Equal(t, []string{"+d", "-a"}, StringSliceDiff([]string{"a", "b", "c"}, []string{"b", "c", "d"}))
}
