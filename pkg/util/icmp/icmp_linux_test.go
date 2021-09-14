//go:build integration && linux
// +build integration,linux

package icmp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/ipv4"

	"github.com/sapcc/kubernikus/pkg/util/netutil"
)

const defaultInterface = "ens192"
const defaultTarget = "cbr0"

func interfaceAddress(t *testing.T, networkInterface string) net.IP {
	t.Helper()
	ip, err := netutil.InterfaceAddress(networkInterface)
	require.NoError(t, err)
	return ip
}

func TestRedirect(t *testing.T) {

	sourceIP := interfaceAddress(t, defaultInterface)

	listener, err := NewListener(sourceIP.String())
	require.NoError(t, err)
	require.NoError(t, listener.SendEcho(&net.IPAddr{IP: interfaceAddress(t, defaultTarget)}, 1))

	msg, err := listener.Read(1 * time.Second)
	require.NoError(t, err)
	require.Equal(t, ipv4.ICMPTypeRedirect, msg.Type)
	require.Equal(t, sourceIP.String(), msg.Body.(*Redirect).NextHop.String())
}
