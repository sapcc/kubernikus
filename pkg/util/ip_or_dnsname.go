package util

import (
	"encoding/json"
	"fmt"
	"net"
)

type IPOrDNSName struct {
	Type       IPOrDNSNameType `json:"-"`
	IPVal      net.IP          `json:"ip"`
	DNSNameVal string          `json:"name"`
}

type IPOrDNSNameType int

const (
	IPType IPOrDNSNameType = iota
	DNSNameType
)

func (s *IPOrDNSName) UnmarshalJSON(value []byte) error {
	if len(value) < 3 {
		return fmt.Errorf(`Parse error: insufficient length for value [%s]`, string(value))
	}
	if s.IPVal = net.ParseIP(string(value[1 : len(value)-1])); s.IPVal != nil {
		s.Type = IPType
		return nil
	}
	s.Type = DNSNameType
	return json.Unmarshal(value, &s.DNSNameVal)
}
