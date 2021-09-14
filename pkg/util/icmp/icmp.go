package icmp

import (
	"math/rand"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	ProtocolIP       = 0  // IPv4 encapsulation, pseudo protocol number
	ProtocolICMP     = 1  //Internet Control Message
	ProtocolIPv6ICMP = 58 // ICMP for IPv6
)

var random = rand.New(rand.NewSource(int64(os.Getpid())))

type Message struct {
	*icmp.Message
	Peer net.Addr
}

type Listener struct {
	c *icmp.PacketConn
	//We use a dedicated socket for sending because we need to set
	//SO_BINDTODEVICE on it
	send net.PacketConn

	ID int
}

func NewListener(address string) (*Listener, error) {
	c, err := icmp.ListenPacket("ip4:icmp", address)
	if err != nil {
		return nil, err
	}

	sendingSocket, err := createSendingSocket(address)
	if err != nil {
		return nil, err
	}

	return &Listener{c, sendingSocket, random.Intn(1 << 16)}, nil
}

func (l *Listener) SetReadDeadline(time time.Time) error {
	return l.c.SetReadDeadline(time)
}
func (l *Listener) Read() (*Message, error) {

	rb := make([]byte, 1500)

	n, peer, err := l.c.ReadFrom(rb)
	if err != nil {
		return nil, err
	}

	//do we need todo ipv4Payload(rb)?
	msg, err := icmp.ParseMessage(ProtocolICMP, rb[:n])
	if err != nil {
		return nil, err
	}

	switch msg.Type {
	case ipv4.ICMPTypeRedirect:
		msg.Body, err = parseRedirectMessageBody(ProtocolICMP, msg.Body.(*icmp.RawBody).Data)
	}

	return &Message{Message: msg, Peer: peer}, err

}

func (l *Listener) SendEcho(addr *net.IPAddr, seq int) error {
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: l.ID & 0xffff, Seq: seq,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}

	_, err = l.send.WriteTo(wb, addr)
	return err
}

func (l *Listener) Close() error {
	err := l.send.Close()
	err2 := l.c.Close()
	if err != nil {
		return err
	}
	return err2
}

func createSendingSocket(address string) (net.PacketConn, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, ProtocolICMP)
	if err != nil {
		return nil, err
	}
	if err := configureSocket(s, address); err != nil {
		return nil, err
	}

	f := os.NewFile(uintptr(s), "schmeicmp")
	return net.FilePacketConn(f)
}
