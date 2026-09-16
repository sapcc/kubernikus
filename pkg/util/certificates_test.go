package util

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

func TestSliceDiff(t *testing.T) {

	cases := []struct {
		Old    []interface{}
		New    []interface{}
		Result []string
	}{
		{
			Old:    []interface{}{"a", "b", "c"},
			New:    []interface{}{"b", "c", "d"},
			Result: []string{"+d", "-a"},
		},
		{
			Old:    []interface{}{"a", net.IPv4(1, 1, 1, 1)},
			New:    []interface{}{"b", net.IPv4(2, 2, 2, 2)},
			Result: []string{"+b", "+2.2.2.2", "-a", "-1.1.1.1"},
		},
	}

	for n, c := range cases {
		assert.Equal(t, c.Result, SliceDiff(c.Old, c.New), "Test case %d failed", n)
	}

	assert.Equal(t, []string{"+2.2.2.2", "-1.1.1.1"}, IPSliceDiff([]net.IP{net.IPv4(1, 1, 1, 1)}, []net.IP{net.IPv4(2, 2, 2, 2)}))
	assert.Equal(t, []string{"+d", "-a"}, StringSliceDiff([]string{"a", "b", "c"}, []string{"b", "c", "d"}))
}

func TestLoadOrCreateCA_Rotation(t *testing.T) {
	kluster := &v1.Kluster{}
	kluster.Name = "test-kluster"

	// Create an initial CA
	var cert, key string
	updates := []CertUpdates{}
	bundle, err := loadOrCreateCA(kluster, "Test", &cert, &key, false, &updates)
	assert.NoError(t, err)
	assert.NotEmpty(t, cert)
	assert.NotEmpty(t, key)
	assert.Len(t, updates, 1) // created

	originalNotAfter := bundle.Certificate.NotAfter
	originalPublicKey := bundle.PrivateKey.Public()
	originalSubject := bundle.Certificate.RawSubject

	// Wait one second so the rotated cert's NotAfter (second precision in ASN.1)
	// is strictly after the original cert's NotAfter.
	time.Sleep(time.Second)

	// Rotate: should produce a new cert with the same key but fresh expiry
	updates2 := []CertUpdates{}
	bundle2, err := loadOrCreateCA(kluster, "Test", &cert, &key, true, &updates2)
	assert.NoError(t, err)
	assert.Len(t, updates2, 1) // rotated

	assert.True(t, bundle2.Certificate.NotAfter.After(originalNotAfter),
		"rotated cert should have a later expiry")
	assert.Equal(t, originalPublicKey, bundle2.PrivateKey.Public(),
		"private key must not change")
	assert.Equal(t, originalSubject, bundle2.Certificate.RawSubject,
		"subject must not change")
	assert.Equal(t, bundle.Certificate.SubjectKeyId, bundle2.Certificate.SubjectKeyId,
		"SubjectKeyId must be identical so existing leaf AuthorityKeyId still matches")
}

func TestEnsure_CARotation(t *testing.T) {
	kluster := &v1.Kluster{}
	kluster.Name = "test-kluster"
	kluster.Spec.AdvertiseAddress = "1.2.3.4"
	kluster.Spec.ServiceCIDR = "198.18.128.0/17"

	store := &v1.Certificates{}
	domain := "example.com"

	// First Ensure — creates all CAs and leaf certs
	factory := NewCertificateFactory(kluster, store, domain)
	updates, err := factory.Ensure(false)
	assert.NoError(t, err)
	assert.NotEmpty(t, updates)

	// Wait one second so the rotated cert's NotAfter (second precision in ASN.1)
	// is strictly after the original cert's NotAfter.
	time.Sleep(time.Second)

	// Capture original CA NotAfter and public keys
	tlsBlock, _ := pem.Decode([]byte(store.TLSCACertificate))
	tlsCert, _ := x509.ParseCertificate(tlsBlock.Bytes)
	origNotAfter := tlsCert.NotAfter
	origPubKey := tlsCert.PublicKey

	// Rotate — must replace all CA certs, same keys
	updates2, err := factory.Ensure(true)
	assert.NoError(t, err)
	// All 9 CAs should appear in updates
	caUpdates := 0
	for _, u := range updates2 {
		if u.Type == "CA certificate" {
			caUpdates++
		}
	}
	assert.Equal(t, 9, caUpdates)

	tlsBlock2, _ := pem.Decode([]byte(store.TLSCACertificate))
	tlsCert2, _ := x509.ParseCertificate(tlsBlock2.Bytes)
	assert.True(t, tlsCert2.NotAfter.After(origNotAfter))
	assert.Equal(t, origPubKey, tlsCert2.PublicKey)
}
