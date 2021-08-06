package kluster

import (
	"net"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
)

//MatchRule checks if input is matched by rule
func MatchRule(input rules.SecGroupRule, rule rules.SecGroupRule) bool {

	if input.Direction != rule.Direction {
		return false
	}
	if input.EtherType != rule.EtherType {
		return false
	}
	if input.RemoteGroupID != rule.RemoteGroupID {
		return false
	}

	if rule.Protocol != "" && input.Protocol != rule.Protocol {
		return false
	}
	if (rule.PortRangeMin > 0 && rule.PortRangeMax > 0) && (input.PortRangeMin < rule.PortRangeMin || input.PortRangeMax > rule.PortRangeMax) {
		return false
	}

	if input.RemoteIPPrefix != rule.RemoteIPPrefix {
		if rule.RemoteIPPrefix != "" {
			_, rulenet, err := net.ParseCIDR(rule.RemoteIPPrefix)
			if err != nil {
				return false
			}
			inputnet := &net.IPNet{IP: make([]byte, 4), Mask: make([]byte, 4)}
			if input.RemoteIPPrefix != "" {
				if _, inputnet, err = net.ParseCIDR(input.RemoteIPPrefix); err != nil {
					return false
				}
			}
			if !CIDRIncluded(inputnet, rulenet) {
				return false
			}
		}
	}
	return true

}

func CIDRIncluded(subject, cidr *net.IPNet) bool {

	lastIP := make([]byte, len(subject.IP))
	for i := range lastIP {
		lastIP[i] = subject.IP[i] | ^subject.Mask[i]
	}
	return cidr.Contains(subject.IP) && cidr.Contains(lastIP)

}
