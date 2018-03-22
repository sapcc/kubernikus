package main

import (
	"fmt"
	"log"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
)

// TestCreateCluster tests kluster creation and waits until the kluster is ready
func (s *E2ETestSuite) TestCreateCluster() {

	log.Printf("Testing kluster creation. Creating kluster %v/%v", s.ProjectName, s.ClusterName)

	kl, err := s.kubernikusClient.Operations.CreateCluster(
		operations.NewCreateClusterParams().WithBody(newE2ECluster(s.ClusterName)),
		s.authFunc(),
	)
	s.handleError(err)

	kluster := kl.Payload
	s.ClusterName = kluster.Name

	err = s.waitForCluster(
		s.ClusterName,
		fmt.Sprintf("Cluster %s wasn't created in time", s.ClusterName),
		func(k *models.Kluster, err error) bool {
			if err != nil {
				log.Println(err)
				return false
			}
			if k.Status.Phase == models.KlusterPhaseRunning {
				log.Printf("Cluster %v is running", k.Name)
				if isNodesHealthyAndRunning(k.Status.NodePools, k.Spec.NodePools) {
					log.Printf("All nodes of cluster %v are healthy and running", k.Name)
					return true
				}
			} else {
				log.Printf("Cluster %v not ready. In phase %v", k.Name, k.Status.Phase)
			}
			return false
		},
	)
	s.handleError(err)

	assert.Equal(s.testing, fmt.Sprintf("%v-%v", s.ClusterName, s.ProjectName), kluster.Name)
	assert.Equal(s.testing, 2, len(kluster.Spec.NodePools))

	for _, node := range kluster.Spec.NodePools {
		assert.Equal(s.testing, "coreos-stable-amd64", node.Image)
	}
}

// TestListClusters tests listing available klusters
func (s *E2ETestSuite) TestListClusters() {

	log.Print("Testing listCluster")

	clusterList, err := s.kubernikusClient.Operations.ListClusters(
		nil,
		s.authFunc(),
	)
	s.handleError(err)

	assert.NotEmpty(s.testing, clusterList.Payload)

	for _, cluster := range clusterList.Payload {
		log.Printf("Found kluster %v", cluster.Name)
		assert.NotNil(s.testing, &cluster)
	}
}

// TestShowCluster tests showing a kluster
func (s *E2ETestSuite) TestShowCluster() {

	log.Print("Testing showCluster")

	cluster, err := s.kubernikusClient.Operations.ShowCluster(
		operations.NewShowClusterParams().WithName(s.ClusterName),
		s.authFunc(),
	)
	s.handleError(err)

	log.Printf("Got kluster %v", cluster.Payload.Name)

	assert.NotEqual(s.testing, "bogusName", cluster.Payload.Name)
	assert.Equal(s.testing, s.ClusterName, cluster.Payload.Name)

}

func (s *E2ETestSuite) TestGetClusterInfo() {

	log.Print("Testing cluster info")

	clusterInfo, err := s.kubernikusClient.Operations.GetClusterInfo(
		operations.NewGetClusterInfoParams().WithName(s.ClusterName),
		s.authFunc(),
	)
	s.handleError(err)

	assert.NotNil(s.testing, clusterInfo.Payload.SetupCommand)
	for _, v := range clusterInfo.Payload.Binaries {
		assert.NotNil(s.testing, v)
	}
}

// TestTerminateCluster tests kluster deletion and waits until the kluster is gone
func (s *E2ETestSuite) TestTerminateCluster() {
	log.Printf("Testing kluster termination")

	_, err := s.kubernikusClient.Operations.TerminateCluster(
		operations.NewTerminateClusterParams().WithName(s.ClusterName),
		s.authFunc(),
	)
	s.handleError(err)

	err = s.waitForCluster(
		s.ClusterName,
		fmt.Sprintf("Cluster %s wasn't terminated in time", s.ClusterName),
		func(k *models.Kluster, err error) bool {
			if err != nil {
				switch err.(type) {
				case *operations.ShowClusterDefault:
					result := err.(*operations.ShowClusterDefault)
					if result.Code() == 404 {
						log.Printf("Cluster %v was terminated", s.ClusterName)
						return true
					}
				case error:
					log.Printf("Failed to show cluster %v: %v", s.ClusterName, err)
					return false
				}
			}
			log.Printf("Cluster %v was not terminated. In phase %v", s.ClusterName, k.Status.Phase)
			return false
		},
	)
	s.handleError(err)
}

// TestUpdateCluster tests updating a kluster
func (s *E2ETestSuite) TestUpdateCluster() {

	log.Printf("Testing kluster update")

	k := newE2ECluster(s.ClusterName)
	nodePoolItem := mediumNodePoolItem()

	k.Spec.NodePools = append(k.Spec.NodePools, *nodePoolItem)

	c, err := s.kubernikusClient.Operations.UpdateCluster(
		operations.NewUpdateClusterParams().WithName(s.ClusterName).WithBody(k),
		s.authFunc(),
	)
	s.handleError(err)

	kluster := c.Payload

	err = s.waitForCluster(
		s.ClusterName,
		fmt.Sprintf("Cluster %s wasn't updated in time", s.ClusterName),
		func(k *models.Kluster, err error) bool {
			if err != nil {
				log.Println(err)
				return false
			}
			if isNodePoolsUpdated(k.Spec.NodePools) && isNodesHealthyAndRunning(k.Status.NodePools, k.Spec.NodePools) {
				log.Printf("Cluster %v was updated and all nodes are healthy and running", k.Name)
				return true
			}
			log.Printf("Cluster %v was not updated or not all nodes are healthy and running", k.Name)
			return false
		},
	)
	s.handleError(err)

	assert.Equal(s.testing, s.ClusterName, kluster.Name)
	assert.Equal(s.testing, 3, len(kluster.Spec.NodePools))
	assert.Contains(s.testing, kluster.Spec.NodePools, nodePoolItem)

}

// TestGetClusterCredentialsAndCreateClientSetForCluster tests getting credentials for a kluster
func (s *E2ETestSuite) TestGetClusterCredentials() {

	log.Print("Testing get kluster credentials")

	cred, err := s.kubernikusClient.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(s.ClusterName),
		s.authFunc(),
	)
	s.handleError(err)

	assert.NotNil(s.testing, cred.Payload.Kubeconfig)
	assert.Contains(s.testing, "clusters", cred.Payload.Kubeconfig)
}
