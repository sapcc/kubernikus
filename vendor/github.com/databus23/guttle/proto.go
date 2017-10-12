package guttle

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

var preambleLength = binary.Size(Header{})

func init() {
	if preambleLength < 0 {
		panic("preambleLength invalid")
	}
}
func newHeader(ip net.IP, port uint16) Header {
	p := Header{DestPort: port}
	copy(p.DestIP[:], ip[0:4])
	return p
}

//Header contains metadata about tunnel requests.
type Header struct {
	DestIP   [4]byte
	DestPort uint16
}

//DestinationIP returns the destination ip for a tunnel request.
func (p Header) DestinationIP() net.IP {
	return []byte{p.DestIP[0], p.DestIP[1], p.DestIP[2], p.DestIP[3]}
}

//DestinationPort returns the destination port for a tunnle request.
func (p Header) DestinationPort() int {
	return int(p.DestPort)
}

func readHeader(reader io.Reader) (hdr Header, err error) {
	data := make([]byte, preambleLength)
	n, err := reader.Read(data)
	if err != nil {
		return
	}
	if n != preambleLength {
		return hdr, errors.New("Not enough bytes to read preamble")
	}
	err = binary.Read(bytes.NewReader(data), binary.BigEndian, &hdr)
	return
}

func writeHeader(writer io.Writer, ip net.IP, port uint16) error {
	header := newHeader(ip, port)
	return binary.Write(writer, binary.BigEndian, &header)
}
