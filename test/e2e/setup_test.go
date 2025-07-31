package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	KlusterPhaseBecomesCreatingTimeout = 5 * time.Minute
	KlusterPhaseBecomesRunningTimeout  = 15 * time.Minute
)

type SetupTests struct {
	Kubernikus  *framework.Kubernikus
	OpenStack   *framework.OpenStack
	KlusterName string
	Reuse       bool
	Dex         bool
	Dashboard   bool
}

func (s *SetupTests) Run(t *testing.T) {
	if s.Reuse == false {
		created := t.Run("Cluster/Create", s.CreateCluster)
		require.True(t, created, "The Kluster must have been created")

		t.Run("Cluster/BecomesCreating", s.KlusterPhaseBecomesCreating)
	}

	running := t.Run("Cluster/BecomesRunning", s.KlusterPhaseBecomesRunning)
	require.True(t, running, "The Kluster must be Running")
}

func (s *SetupTests) CreateCluster(t *testing.T) {
	version := "1.25.5"
	if v := os.Getenv("KLUSTER_VERSION"); v != "" {
		version = v
	}

	clusterCidr := "100.100.0.0/16"
	if cidr := os.Getenv("KLUSTER_CIDR"); cidr != "" {
		clusterCidr = cidr
	}

	osImages := []string{"flatcar-stable-amd64"}
	if image := os.Getenv("KLUSTER_OS_IMAGES"); image != "" {
		osImages = strings.Split(image, ",")
	}
	require.LessOrEqual(t, len(osImages), SmokeTestNodeCount, "more os images then smoke test node specified")

	flavor := "c_c2_m2"
	if os.Getenv("KLUSTER_FLAVOR") != "" {
		flavor = os.Getenv("KLUSTER_FLAVOR")
	}
	customRootDiskSize := 0 // no custom root disk size by default
	if os.Getenv("KLUSTER_CUSTOM_ROOT_DISK_SIZE") != "" {
		var err error
		customRootDiskSize, err = strconv.Atoi(os.Getenv("KLUSTER_CUSTOM_ROOT_DISK_SIZE"))
		require.NoError(t, err, "KLUSTER_CUSTOM_ROOT_DISK_SIZE must be a valid integer")
		require.Greater(t, customRootDiskSize, 0, "KLUSTER_CUSTOM_ROOT_DISK_SIZE must be greater than 0")
	}

	pools := []models.NodePool{}
	for i, image := range osImages {
		pools = append(pools, models.NodePool{
			Name:               fmt.Sprintf("pool%d", i+1),
			Flavor:             flavor,
			Size:               1,
			AvailabilityZone:   os.Getenv("NODEPOOL_AVZ"),
			Image:              image,
			Labels:             []string{"image=" + image},
			CustomRootDiskSize: int64(customRootDiskSize),
		})
	}
	//we fill up the first pool in case the number of images is smaller then the  smoke test node count
	pools[0].Size = int64(SmokeTestNodeCount - (len(pools) - 1))

	kluster := &models.Kluster{
		Name: s.KlusterName,
		Spec: models.KlusterSpec{
			Version:      version,
			SSHPublicKey: os.Getenv("KLUSTER_SSH_PUBLIC_KEY"),
			ClusterCIDR:  &clusterCidr,
			NodePools:    pools,
			Openstack: models.OpenstackSpec{
				RouterID: os.Getenv("KLUSTER_ROUTER"),
			},
			Dex:       &s.Dex,
			Dashboard: &s.Dashboard,
		},
	}

	cluster, err := s.Kubernikus.Client.Operations.CreateCluster(
		operations.NewCreateClusterParams().WithBody(kluster),
		s.Kubernikus.AuthInfo,
	)

	require.NoError(t, err, "There should be no error")
	require.NotNil(t, cluster.Payload, "There should be payload")
	assert.Equalf(t, s.KlusterName, cluster.Payload.Name, "There should be a kluster with name: %v", s.KlusterName)
	assert.Equal(t, cluster.Payload.Status.Phase, models.KlusterPhasePending, "Kluster should be in phase Pending")
}

func (s *SetupTests) KlusterPhaseBecomesCreating(t *testing.T) {
	phase, err := s.Kubernikus.WaitForKlusterPhase(s.KlusterName, models.KlusterPhaseCreating, KlusterPhaseBecomesCreatingTimeout)

	if assert.NoError(t, err, "There should be no error") {
		assert.Equal(t, phase, models.KlusterPhaseCreating, "Kluster should become Creating")
	}
}

func (s *SetupTests) KlusterPhaseBecomesRunning(t *testing.T) {
	phase, err := s.Kubernikus.WaitForKlusterPhase(s.KlusterName, models.KlusterPhaseRunning, KlusterPhaseBecomesRunningTimeout)

	if assert.NoError(t, err, "There should be no error") {
		require.Equal(t, phase, models.KlusterPhaseRunning, "Kluster should become Running")
	}
}
