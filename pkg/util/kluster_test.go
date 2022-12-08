package util

import (
	"testing"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func TestKlusterNeedsUpgrade(t *testing.T) {

	cases := []struct {
		from     string
		to       string
		expected bool
	}{
		{from: "1.10.1", to: "1.10.2", expected: true},
		{from: "1.10.1", to: "1.11.2", expected: true},
		{from: "1.10.1", to: "1.12.2", expected: false},
		{from: "1.10.2", to: "1.10.1", expected: true},
		{from: "1.10.1", to: "1.9.2", expected: false},
		{from: "1.10.1", to: "1.10.1", expected: false},
	}

	for _, c := range cases {
		k := new(v1.Kluster)
		k.Spec.Version = c.to
		k.Status.ApiserverVersion = c.from
		if result, err := KlusterNeedsUpgrade(k); err != nil || result != c.expected {
			t.Errorf("expected %t for %s --> %s, err: %s", c.expected, c.from, c.to, err)
		}
	}
}
