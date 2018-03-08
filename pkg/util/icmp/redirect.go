package icmp

import (
	"errors"
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var errMessageTooShort = errors.New("message too short")

type Redirect struct {
	NextHop net.IP
	Header  *ipv4.Header
	Data    []byte // data
}

func (r *Redirect) Len(proto int) int {
	if r == nil {
		return 0
	}
	return 4 + r.Header.Len + len(r.Data)
}

func (r *Redirect) Marshal(proto int) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func parseRedirectMessageBody(proto int, b []byte) (icmp.MessageBody, error) {
	bodyLen := len(b)
	if bodyLen < 4 {
		return nil, errMessageTooShort
	}

	p := &Redirect{NextHop: net.IPv4(b[0], b[1], b[2], b[3])}
	header, err := ipv4.ParseHeader(b[4:])
	if err != nil {
	}
	p.Header = header
	p.Data = b[4+header.Len:]

	return p, nil
}
