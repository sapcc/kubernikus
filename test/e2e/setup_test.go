package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

const (
	KlusterPhaseBecomesCreatingTimeout = 1 * time.Minute
	KlusterPhaseBecomesRunningTimeout  = 15 * time.Minute
)

type SetupTests struct {
	Kubernikus  *framework.Kubernikus
	OpenStack   *framework.OpenStack
	KlusterName string
	Reuse       bool
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
	kluster := &models.Kluster{
		Name: s.KlusterName,
		Spec: models.KlusterSpec{
			Version:      "1.19.4",
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCXIxVEUgtUVkvk2VM1hmIb8MxvxsmvYoiq9OBy3J8akTGNybqKsA2uhcwxSJX5Cn3si8kfMfka9EWiJT+e1ybvtsGILO5XRZPxyhYzexwb3TcALwc3LuzpF3Z/Dg2jYTRELTGhYmyca3mxzTlCjNXvYayLNedjJ8fIBzoCuSXNqDRToHru7h0Glz+wtuE74mNkOiXSvhtuJtJs7VCNVjobFQNfC1aeDsri2bPRHJJZJ0QF4LLYSayMEz3lVwIDyAviQR2Aa97WfuXiofiAemfGqiH47Kq6b8X7j3bOYGBvJKMUV7XeWhGsskAmTsvvnFxkc5PAD3Ct+liULjiQWlzDrmpTE8aMqLK4l0YQw7/8iRVz6gli42iEc2ZG56ob1ErpTLAKFWyCNOebZuGoygdEQaGTIIunAncXg5Rz07TdPl0Tf5ZZLpiAgR5ck0H1SETnjDTZ/S83CiVZWJgmCpu8YOKWyYRD4orWwdnA77L4+ixeojLIhEoNL8KlBgsP9Twx+fFMWLfxMmiuX+yksM6Hu+Lsm+Ao7Q284VPp36EB1rxP1JM7HCiEOEm50Jb6hNKjgN4aoLhG5yg+GnDhwCZqUwcRJo1bWtm3QvRA+rzrGZkId4EY3cyOK5QnYV5+24x93Ex0UspHMn7HGsHUESsVeV0fLqlfXyd2RbHTmDMP6w== Kubernikus Master Key",
			NodePools: []models.NodePool{
				{
					Name:             "small",
					Flavor:           "m1.small",
					Size:             SmokeTestNodeCount,
					AvailabilityZone: os.Getenv("NODEPOOL_AVZ"),
				},
			},
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
