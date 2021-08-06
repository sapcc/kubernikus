package kluster

import (
	"testing"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/stretchr/testify/assert"
)

func TestMatchRule(t *testing.T) {

	cases := []struct {
		Subject rules.SecGroupRule
		Rule    rules.SecGroupRule
		Result  bool
	}{
		{rules.SecGroupRule{}, rules.SecGroupRule{}, true},
		{rules.SecGroupRule{Direction: "ingress"}, rules.SecGroupRule{Direction: "egress"}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, rules.SecGroupRule{Protocol: "udp"}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, rules.SecGroupRule{Direction: "ingress", Protocol: "udp"}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, true},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 80, PortRangeMax: 0}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, true},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 80, PortRangeMax: 80}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 79, PortRangeMax: 80}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 80, PortRangeMax: 80}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 80, PortRangeMax: 81}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 80, PortRangeMax: 80}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 80, PortRangeMax: 80}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", PortRangeMin: 80, PortRangeMax: 80}, true},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.0.0/24"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, true},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.0.0/24"}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.0.0/25"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.0.0/24"}, true},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.0.0/23"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.0.0/24"}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.0.0/23"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.1.0/24"}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.2.0/23"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.2.0/24"}, false},
		{rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.2.0/24"}, rules.SecGroupRule{Direction: "ingress", Protocol: "tcp", RemoteIPPrefix: "10.0.2.0/23"}, true},
	}

	for _, c := range cases {
		assert.Equalf(t, c.Result, MatchRule(c.Subject, c.Rule), "Expect MatchRule to return %v for input %v and rule %v", c.Result, c.Subject, c.Rule)
	}

}
