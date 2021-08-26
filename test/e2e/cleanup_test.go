package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	blockstorage_quota "github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/quotasets"
	compute_quota "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/quotasets"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	KlusterPhaseBecomesTerminatingTimeout = 1 * time.Minute
	WaitForKlusterToBeDeletedTimeout      = 15 * time.Minute
)

type CleanupTests struct {
	Kubernikus  *framework.Kubernikus
	OpenStack   *framework.OpenStack
	KlusterName string
	Reuse       bool
}

func (s *CleanupTests) Run(t *testing.T) {
	if t.Run("Cluster/Terminate", s.TerminateCluster) {
		t.Run("Cluster/BecomesTerminating", s.KlusterPhaseBecomesTerminating)
		t.Run("Cluster/IsDeleted", s.WaitForKlusterToBeDeleted)

		if s.Reuse == false {
			t.Run("QuotaPostFlightCheck", s.QuotaPostFlightCheck)
			t.Run("ServerGroupsGotDeleted", s.ServerGroupsGotDeleted)
			t.Run("LoadbalancerGotDeleted", s.LoadbalancerGotDeleted)
		}
	}
}

func (s *CleanupTests) TerminateCluster(t *testing.T) {
	_, err := s.Kubernikus.Client.Operations.TerminateCluster(
		operations.NewTerminateClusterParams().WithName(s.KlusterName),
		s.Kubernikus.AuthInfo,
	)
	assert.NoError(t, err, "There should be no error")
}

func (s *CleanupTests) WaitForKlusterToBeDeleted(t *testing.T) {
	err := s.Kubernikus.WaitForKlusterToBeDeleted(s.KlusterName, WaitForKlusterToBeDeletedTimeout)
	require.NoError(t, err, "There should be no error while waiting %v for the kluster to be deleted", WaitForKlusterToBeDeletedTimeout)
}

func (s *CleanupTests) KlusterPhaseBecomesTerminating(t *testing.T) {
	phase, err := s.Kubernikus.WaitForKlusterPhase(s.KlusterName, models.KlusterPhaseTerminating, KlusterPhaseBecomesTerminatingTimeout)

	if assert.NoError(t, err, "There should be no error") {
		assert.Equal(t, phase, models.KlusterPhaseTerminating, "Kluster should become Terminating")
	}
}

func (s *CleanupTests) QuotaPostFlightCheck(t *testing.T) {
	project, err := tokens.Get(s.OpenStack.Identity, s.OpenStack.Provider.Token()).ExtractProject()
	require.NoError(t, err, "There should be no error while getting project from token")
	require.NotNil(t, project, "project returned from Token %s was nil. WTF?", s.OpenStack.Provider.Token())

	var computeQ compute_quota.QuotaDetailSet
	var storageQ blockstorage_quota.QuotaUsageSet
	err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
		var err error
		if computeQ, err = compute_quota.GetDetail(s.OpenStack.Compute, project.ID).Extract(); err != nil {
			return false, fmt.Errorf("Failed to fetch compute usage: %w", err)
		}
		if storageQ, err = blockstorage_quota.GetUsage(s.OpenStack.BlockStorage, project.ID).Extract(); err != nil {
			return false, fmt.Errorf("Failed to fetch block storage usage: %w", err)
		}
		return computeQ.Cores.InUse == 0 && computeQ.Instances.InUse == 0 && computeQ.RAM.InUse == 0 && storageQ.Volumes.InUse == 0 && storageQ.Gigabytes.InUse == 0, nil
	})

	require.NoError(t, err, "There should be no error while getting quota/usage details")

	assert.Zero(t, computeQ.Cores.InUse, "There should be no cores left in use")
	assert.Zero(t, computeQ.Instances.InUse, "There should be no instances left in use")
	assert.Zero(t, computeQ.RAM.InUse, "There should be no RAM left in use")
	assert.Zero(t, storageQ.Volumes.InUse, "There should be no Volume left in use")
	assert.Zero(t, storageQ.Gigabytes.InUse, "There should be no Storage left in use")
}

func (s *CleanupTests) ServerGroupsGotDeleted(t *testing.T) {
	computeClient, err := openstack.NewComputeV2(s.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "There should be no error creating compute client")

	allPages, err := servergroups.List(computeClient, servergroups.ListOpts{}).AllPages()
	require.NoError(t, err, "There should be no error listing server groups")

	allGroups, err := servergroups.ExtractServerGroups(allPages)
	require.NoError(t, err, "There should be no error extracting server groups")

	count := 0
	for _, sg := range allGroups {
		if strings.HasPrefix(sg.Name, "e2e-") {
			count++
		}
	}
	require.Equal(t, 0, count, "There should be no server groups left")
}

func (s *CleanupTests) LoadbalancerGotDeleted(t *testing.T) {
	lbClient, err := openstack.NewLoadBalancerV2(s.OpenStack.Provider, gophercloud.EndpointOpts{})
	require.NoError(t, err, "There should be no error getting a loadbalancer client")

	allPages, err := loadbalancers.List(lbClient, loadbalancers.ListOpts{}).AllPages()
	require.NoError(t, err, "There should be no error while listing loadbalancers")

	allLoadbalancers, err := loadbalancers.ExtractLoadBalancers(allPages)
	require.NoError(t, err, "There should be no error while extracting loadbalancers")

	count := 0
	for _, lb := range allLoadbalancers {
		if strings.HasSuffix(lb.Name, "_e2e-lb") {
			count++
		}
	}

	require.Equal(t, 0, count, "There should be no Loadbalancers left")
}
