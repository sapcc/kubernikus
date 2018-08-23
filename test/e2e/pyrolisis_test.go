package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

type PyrolisisTests struct {
	Kubernikus *framework.Kubernikus
	Reuse      bool
}

func (p *PyrolisisTests) Run(t *testing.T) {
	if p.Reuse == false {
		quota := t.Run("SettingKlustersOnFire", p.SettingKlustersOnFire)
		require.True(t, quota, "Klusters must burn")

		t.Run("Wait", func(t *testing.T) {
			t.Run("Klusters", p.WaitForE2EKlustersTerminated)
		})
	}
}

func (p *PyrolisisTests) SettingKlustersOnFire(t *testing.T) {
	res, err := p.Kubernikus.Client.Operations.ListClusters(
		operations.NewListClustersParams(),
		p.Kubernikus.AuthInfo,
	)
	require.NoError(t, err, "There should be no error while listing klusters")

	for _, kluster := range res.Payload {
		if strings.HasPrefix(kluster.Name, "e2e-") {
			t.Run(fmt.Sprintf("TerminatingKluster-%v", kluster.Name), func(t *testing.T) {
				_, err := p.Kubernikus.Client.Operations.TerminateCluster(
					operations.NewTerminateClusterParams().WithName(kluster.Name),
					p.Kubernikus.AuthInfo,
				)
				assert.NoError(t, err, "There should be no error while terminating klusters")
			})
		}
	}
}

func (p *PyrolisisTests) WaitForE2EKlustersTerminated(t *testing.T) {
	err := p.Kubernikus.WaitForKlusters("e2e-", 0, WaitForKlusterToBeDeletedTimeout)
	assert.NoError(t, err, "E2E Klusters didn't burn down in time")
}
