package ip

import (
	"net"
)

//adapted from  k8s.io/pkg/controller/route
func CIDROverlap(cidr1, cidr2 *net.IPNet) bool {

	lastIP1 := make([]byte, len(cidr1.IP))
	for i := range lastIP1 {
		lastIP1[i] = cidr1.IP[i] | ^cidr1.Mask[i]
	}
	lastIP2 := make([]byte, len(cidr2.IP))
	for i := range lastIP2 {
		lastIP2[i] = cidr2.IP[i] | ^cidr2.Mask[i]
	}

	if cidr2.Contains(cidr1.IP) || cidr2.Contains(lastIP1) || cidr1.Contains(cidr2.IP) || cidr1.Contains(lastIP2) {
		return true
	}
	return false
}
