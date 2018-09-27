package main

import (
	"testing"

	blockstorage_quota "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/quotasets"
	compute_quota "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/quotasets"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/test/e2e/framework"
)

type PreFlightTests struct {
	Kubernikus *framework.Kubernikus
	OpenStack  *framework.OpenStack
	Reuse      bool
}

func (s *PreFlightTests) Run(t *testing.T) {
	if s.Reuse == false {
		quota := t.Run("Quota", s.QuotaPreflightCheck)
		require.True(t, quota, "The Kluster must have enough quota")

	}
}

func (s *PreFlightTests) QuotaPreflightCheck(t *testing.T) {
	project, err := tokens.Get(s.OpenStack.Identity, s.OpenStack.Provider.Token()).ExtractProject()
	require.NoError(t, err, "There should be no error while getting project from token")

	quota, err := compute_quota.GetDetail(s.OpenStack.Compute, project.ID).Extract()
	require.NoError(t, err, "There should be no error while getting compute quota details")

	storage, err := blockstorage_quota.GetUsage(s.OpenStack.BlockStorage, project.ID).Extract()
	require.NoError(t, err, "There should be no error while getting storage quota details")

	assert.True(t, quota.Cores.Limit-quota.Cores.InUse >= SmokeTestNodeCount*2, "There should be at least %v cores quota left", SmokeTestNodeCount*2)
	assert.True(t, quota.Instances.Limit-quota.Instances.InUse >= SmokeTestNodeCount, "There should be at least %v instances quota left", SmokeTestNodeCount)
	assert.True(t, quota.RAM.Limit-quota.RAM.InUse >= SmokeTestNodeCount*2048, "There should be at least %v RAM quota left", SmokeTestNodeCount*2048)
	assert.True(t, storage.Volumes.Limit-storage.Volumes.InUse >= 1, "There should be at least %v Volume quota left", 1)
	assert.True(t, storage.Gigabytes.Limit-storage.Gigabytes.InUse >= 1, "There should be at least %v Gigabytes quota left", 1)
}
