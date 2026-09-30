//go:build linux
// +build linux

package nftables_test

import (
	"testing"

	"github.com/google/nftables"
	"github.com/mdlayher/netlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	knftables "github.com/sapcc/kubernikus/pkg/util/nftables"
)

// newTestConn creates a test connection that records netlink messages without
// touching the kernel. This mimics what nfttest.NewConn() would do.
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

func TestSyncRules_CreatesTableAndChain(t *testing.T) {
	conn := newTestConn(t)
	nft := knftables.NewWithConn(conn)

	err := nft.SyncRules([]string{"10.0.0.1/32", "10.244.0.0/16"}, 9191)
	require.NoError(t, err)

	tables, err := conn.ListTables()
	require.NoError(t, err)
	require.Len(t, tables, 1)
	assert.Equal(t, "kubernikus", tables[0].Name)

	chains, err := conn.ListChains()
	require.NoError(t, err)
	require.Len(t, chains, 1)
	assert.Equal(t, "tunnels", chains[0].Name)
}

func TestSyncRules_CreatesOneRulePerCIDR(t *testing.T) {
	conn := newTestConn(t)
	nft := knftables.NewWithConn(conn)

	cidrs := []string{"10.0.0.1/32", "10.244.0.0/16", "10.96.0.0/12"}
	err := nft.SyncRules(cidrs, 9191)
	require.NoError(t, err)

	chains, err := conn.ListChains()
	require.NoError(t, err)
	require.Len(t, chains, 1)

	rules, err := conn.GetRules(chains[0].Table, chains[0])
	require.NoError(t, err)
	assert.Len(t, rules, len(cidrs))
}

func TestSyncRules_IsAtomic(t *testing.T) {
	conn := newTestConn(t)
	nft := knftables.NewWithConn(conn)

	// First sync with 3 CIDRs
	err := nft.SyncRules([]string{"10.0.0.1/32", "10.244.0.0/16", "10.96.0.0/12"}, 9191)
	require.NoError(t, err)

	// Second sync with 1 CIDR — rules must be replaced, not appended
	err = nft.SyncRules([]string{"10.0.0.2/32"}, 9191)
	require.NoError(t, err)

	chains, err := conn.ListChains()
	require.NoError(t, err)
	rules, err := conn.GetRules(chains[0].Table, chains[0])
	require.NoError(t, err)
	assert.Len(t, rules, 1)
}

func TestSyncRules_EmptyCIDRs(t *testing.T) {
	conn := newTestConn(t)
	nft := knftables.NewWithConn(conn)

	err := nft.SyncRules([]string{}, 9191)
	require.NoError(t, err)

	chains, err := conn.ListChains()
	require.NoError(t, err)
	rules, err := conn.GetRules(chains[0].Table, chains[0])
	require.NoError(t, err)
	assert.Len(t, rules, 0)
}

func TestDestroy_RemovesTable(t *testing.T) {
	conn := newTestConn(t)
	nft := knftables.NewWithConn(conn)

	require.NoError(t, nft.SyncRules([]string{"10.0.0.1/32"}, 9191))
	require.NoError(t, nft.Destroy())

	tables, err := conn.ListTables()
	require.NoError(t, err)
	assert.Len(t, tables, 0)
}
