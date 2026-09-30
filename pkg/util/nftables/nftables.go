//go:build linux
// +build linux

package nftables

import (
	"encoding/binary"
	"net"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

// Interface manages nftables redirect rules for the wormhole tunnel.
type Interface interface {
	// SyncRules atomically replaces all redirect rules in the kubernikus table.
	// cidrs must be valid CIDR strings (e.g. "10.0.0.1/32", "10.244.0.0/16").
	// All TCP traffic matching any cidr is redirected to redirectPort on localhost.
	SyncRules(cidrs []string, redirectPort int) error
	// Destroy removes the kubernikus table entirely.
	Destroy() error
}

const (
	tableName = "kubernikus"
	chainName = "tunnels"
)

type runner struct {
	conn *nftables.Conn
}

// New returns an Interface that manages rules via the kernel netlink API.
// Requires CAP_NET_ADMIN.
func New() (Interface, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, err
	}
	return &runner{conn: conn}, nil
}

// NewWithConn returns an Interface using the provided connection.
// Intended for testing with nfttest.NewConn().
func NewWithConn(conn *nftables.Conn) Interface {
	return &runner{conn: conn}
}

func (r *runner) SyncRules(cidrs []string, redirectPort int) error {
	table := r.ensureTable()

	// Flush all existing rules in the table (atomic replace).
	r.conn.FlushTable(table)

	chain := r.conn.AddChain(&nftables.Chain{
		Name:     chainName,
		Table:    table,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityNATDest,
	})

	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		r.conn.AddRule(redirectRule(table, chain, ipNet, redirectPort))
	}

	return r.conn.Flush()
}

func (r *runner) Destroy() error {
	tables, err := r.conn.ListTables()
	if err != nil {
		return err
	}
	for _, t := range tables {
		if t.Name == tableName {
			r.conn.DelTable(t)
			return r.conn.Flush()
		}
	}
	return nil
}

func (r *runner) ensureTable() *nftables.Table {
	return &nftables.Table{
		Name:   tableName,
		Family: nftables.TableFamilyIPv4,
	}
}

// redirectRule builds an nftables rule:
//   tcp daddr <ipNet> redirect to :<redirectPort>
func redirectRule(table *nftables.Table, chain *nftables.Chain, ipNet *net.IPNet, redirectPort int) *nftables.Rule {
	ip := ipNet.IP.To4()
	mask := ipNet.Mask

	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(redirectPort))

	return &nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: []expr.Any{
			// Match TCP protocol
			&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{unix.IPPROTO_TCP},
			},
			// Load destination IP
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseNetworkHeader,
				Offset:       16,
				Len:          4,
			},
			// Apply subnet mask
			&expr.Bitwise{
				SourceRegister: 1,
				DestRegister:   1,
				Len:            4,
				Mask:           []byte(mask),
				Xor:            []byte{0, 0, 0, 0},
			},
			// Compare network address
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte(ip),
			},
			// Set port in register 2
			&expr.Immediate{
				Register: 2,
				Data:     portBytes,
			},
			// REDIRECT using port from register 2
			&expr.Redir{
				RegisterProtoMin: 2,
			},
		},
	}
}
