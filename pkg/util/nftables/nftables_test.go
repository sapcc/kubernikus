//go:build linux

package nftables_test

import (
	"testing"

	"github.com/google/nftables"
	"github.com/mdlayher/netlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	knftables "github.com/sapcc/kubernikus/pkg/util/nftables"
)

// newTestConn creates a connection that ACKs all writes without touching the
// kernel. Read-back calls (ListTables, GetRules) return empty results — state
// verification requires a real kernel (see e2e tests).
func newTestConn(t *testing.T) *nftables.Conn {
	t.Helper()
	c, err := nftables.New(nftables.WithTestDial(
		func(req []netlink.Message) ([]netlink.Message, error) {
			acks := make([]netlink.Message, 0, len(req))
			for _, msg := range req {
				if msg.Header.Flags&netlink.Acknowledge != 0 {
					acks = append(acks, netlink.Message{
						Header: netlink.Header{
							Length:   4,
							Type:     netlink.Error,
							Sequence: msg.Header.Sequence,
							PID:      msg.Header.PID,
						},
						Data: []byte{0, 0, 0, 0},
					})
				}
			}
			return acks, nil
		}))
	require.NoError(t, err)
	return c
}

func TestSyncRules_NoError(t *testing.T) {
	nft := knftables.NewWithConn(newTestConn(t))
	err := nft.SyncRules([]string{"10.0.0.1/32", "10.244.0.0/16", "10.96.0.0/12"}, 9191)
	require.NoError(t, err)
}

func TestSyncRules_EmptyCIDRs_NoError(t *testing.T) {
	nft := knftables.NewWithConn(newTestConn(t))
	err := nft.SyncRules([]string{}, 9191)
	require.NoError(t, err)
}

func TestSyncRules_InvalidCIDR_ReturnsError(t *testing.T) {
	nft := knftables.NewWithConn(newTestConn(t))
	err := nft.SyncRules([]string{"not-a-cidr"}, 9191)
	require.Error(t, err)
}

func TestSyncRules_CalledTwice_NoError(t *testing.T) {
	nft := knftables.NewWithConn(newTestConn(t))
	require.NoError(t, nft.SyncRules([]string{"10.0.0.1/32", "10.244.0.0/16"}, 9191))
	// Second call exercises the FlushTable + re-add path.
	require.NoError(t, nft.SyncRules([]string{"10.0.0.2/32"}, 9191))
}

func TestDestroy_NoError(t *testing.T) {
	nft := knftables.NewWithConn(newTestConn(t))
	require.NoError(t, nft.Destroy())
}

func nftMsgType(t uint16) netlink.HeaderType {
	return netlink.HeaderType(unix.NFNL_SUBSYS_NFTABLES<<8 | t)
}

func TestSyncRules_Twice_SameRuleCount(t *testing.T) {
	type batch struct{ newRules, delRules int }
	var batches []batch
	conn, err := nftables.New(nftables.WithTestDial(
		func(req []netlink.Message) ([]netlink.Message, error) {
			var b batch
			acks := make([]netlink.Message, 0, len(req))
			for _, msg := range req {
				switch msg.Header.Type {
				case nftMsgType(unix.NFT_MSG_NEWRULE):
					b.newRules++
				case nftMsgType(unix.NFT_MSG_DELRULE):
					b.delRules++
				}
				if msg.Header.Flags&netlink.Acknowledge != 0 {
					acks = append(acks, netlink.Message{
						Header: netlink.Header{
							Length:   4,
							Type:     netlink.Error,
							Sequence: msg.Header.Sequence,
							PID:      msg.Header.PID,
						},
						Data: []byte{0, 0, 0, 0},
					})
				}
			}
			batches = append(batches, b)
			return acks, nil
		}))
	require.NoError(t, err)
	nft := knftables.NewWithConn(conn)

	cidrs := []string{"10.0.0.1/32", "10.244.0.0/16", "10.96.0.0/12"}
	require.NoError(t, nft.SyncRules(cidrs, 9191))
	require.NoError(t, nft.SyncRules(cidrs, 9191))

	var got []batch
	for _, b := range batches {
		if b.newRules > 0 || b.delRules > 0 {
			got = append(got, b)
		}
	}
	assert.Equal(t, []batch{{3, 1}, {3, 1}}, got)
}
