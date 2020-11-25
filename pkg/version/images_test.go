package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegionReplacement(t *testing.T) {

	registry := ImageRegistry{
		Versions: map[string]KlusterVersion{
			"1.0.0": {
				Hyperkube: ImageVersion{
					Repository: "something.$REGION.cloud.sap",
					Tag:        "1.0.0",
				},
				Kubelet: ImageVersion{
					Repository: "no.region.cloud.sap",
					Tag:        "2.0.0",
				},
			},
		},
	}

	replaceRegionVarInRepositoryField(registry.Versions, "test")

	require.Equal(t,
		ImageVersion{Repository: "something.test.cloud.sap", Tag: "1.0.0"},
		registry.Versions["1.0.0"].Hyperkube,
	)
	require.Equal(t,
		ImageVersion{Repository: "no.region.cloud.sap", Tag: "2.0.0"},
		registry.Versions["1.0.0"].Kubelet,
	)

}
