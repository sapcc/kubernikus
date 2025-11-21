//go:build integration

package icmp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func TestPing(t *testing.T) {
	listener, err := NewListener("0.0.0.0")
	require.NoError(t, err)

	require.NoError(t, listener.SendEcho(&net.IPAddr{IP: net.ParseIP("8.8.8.8")}, 23))

	msg, err := listener.Read(1 * time.Second)
	require.NoError(t, err)

	require.Equal(t, ipv4.ICMPTypeEchoReply, msg.Type)
	require.Equal(t, listener.ID, msg.Body.(*icmp.Echo).ID)
	require.Equal(t, 23, msg.Body.(*icmp.Echo).Seq)
}
