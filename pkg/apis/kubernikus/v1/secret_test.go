package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestSecretMarsheling(t *testing.T) {

	secret := Secret{
		NodePassword:   "bla",
		BootstrapToken: "blu",
	}

	data, err := secret.ToData()
	require.NoError(t, err)
	newSecret, err := NewSecret(&corev1.Secret{Data: data})
	require.NoError(t, err)
	require.Equal(t, secret, *newSecret)

}
