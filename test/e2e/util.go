package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func (s *E2ETestSuite) waitForCluster(klusterName, errorString string, waitUntilFunc func(k *models.Kluster, err error) bool) error {
	for start := time.Now(); time.Since(start) < Timeout; time.Sleep(CheckInterval) {
		select {
		default:
			kluster, err := s.kubernikusClient.Operations.ShowCluster(
				operations.NewShowClusterParams().WithName(klusterName),
				s.authFunc(),
			)
			// pass an handle err in waitUntilFunc
			if err != nil {
				if waitUntilFunc(nil, err) {
					return nil
				}
			}
			if waitUntilFunc(kluster.Payload, err) {
				return nil
			}
		case <-s.stopCh:
			os.Exit(1)
		}
	}
	return fmt.Errorf(errorString)
}

func isNodePoolsUpdated(nodePools []models.NodePool) bool {
	// check if both nodePools exists and one of the is medium nodePool
	for _, v := range nodePools {
		if v.Name == "medium" && len(nodePools) == 2 {
			return true
		}
	}
	return false
}

func isNodesHealthyAndRunning(nodePoolsStatus []models.NodePoolInfo, nodePoolsSpec []models.NodePool) bool {
	for _, v := range nodePoolsStatus {
		expectedSize := getNodePoolSizeFromSpec(nodePoolsSpec, v.Name)
		if expectedSize == -1 {
			log.Printf("couldn't find nodepool with name %v in spec", v.Name)
			return false
		}
		if v.Healthy != expectedSize || v.Running != expectedSize {
			log.Printf("nodepool %v: expected %v node(s), actual: healthy %v, running %v", v.Name, expectedSize, v.Healthy, v.Running)
			return false
		}
	}
	return true
}

func getNodePoolSizeFromSpec(nodePoolsSpec []models.NodePool, name string) int64 {
	for _, v := range nodePoolsSpec {
		if v.Name == name {
			return v.Size
		}
	}
	return -1
}

func (s *E2ETestSuite) emptyNodePoolsOfKluster() {

	log.Printf("stopping all nodes of cluster %v", s.ClusterName)

	cluster, err := s.kubernikusClient.Operations.ShowCluster(
		operations.NewShowClusterParams().WithName(s.ClusterName),
		s.authFunc(),
	)
	s.handleError(err)

	nodePools := []models.NodePool{}
	for _, nodePool := range cluster.Payload.Spec.NodePools {
		nodePool.Size = 0
		nodePools = append(nodePools, nodePool)
	}
	cluster.Payload.Spec.NodePools = nodePools

	// empty node pools
	_, err = s.kubernikusClient.Operations.UpdateCluster(
		operations.NewUpdateClusterParams().
			WithName(s.ClusterName).
			WithBody(cluster.Payload),
		s.authFunc(),
	)
	s.handleError(err)

	err = s.waitForCluster(
		s.ClusterName,
		fmt.Sprintf("Not all nodes of cluster %v could be terminated in time", s.ClusterName),
		func(k *models.Kluster, err error) bool {
			if err != nil {
				log.Println(err)
				return false
			}
			for _, node := range k.Status.NodePools {
				if node.Running != 0 {
					log.Printf("Cluster %v has nodes in state running", k.Name)
					return false
				}
			}
			return true
		},
	)
	s.handleError(err)
}

func newE2ECluster(klusterName string) *models.Kluster {
	return &models.Kluster{
		Name: klusterName,
		Spec: models.KlusterSpec{
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCXIxVEUgtUVkvk2VM1hmIb8MxvxsmvYoiq9OBy3J8akTGNybqKsA2uhcwxSJX5Cn3si8kfMfka9EWiJT+e1ybvtsGILO5XRZPxyhYzexwb3TcALwc3LuzpF3Z/Dg2jYTRELTGhYmyca3mxzTlCjNXvYayLNedjJ8fIBzoCuSXNqDRToHru7h0Glz+wtuE74mNkOiXSvhtuJtJs7VCNVjobFQNfC1aeDsri2bPRHJJZJ0QF4LLYSayMEz3lVwIDyAviQR2Aa97WfuXiofiAemfGqiH47Kq6b8X7j3bOYGBvJKMUV7XeWhGsskAmTsvvnFxkc5PAD3Ct+liULjiQWlzDrmpTE8aMqLK4l0YQw7/8iRVz6gli42iEc2ZG56ob1ErpTLAKFWyCNOebZuGoygdEQaGTIIunAncXg5Rz07TdPl0Tf5ZZLpiAgR5ck0H1SETnjDTZ/S83CiVZWJgmCpu8YOKWyYRD4orWwdnA77L4+ixeojLIhEoNL8KlBgsP9Twx+fFMWLfxMmiuX+yksM6Hu+Lsm+Ao7Q284VPp36EB1rxP1JM7HCiEOEm50Jb6hNKjgN4aoLhG5yg+GnDhwCZqUwcRJo1bWtm3QvRA+rzrGZkId4EY3cyOK5QnYV5+24x93Ex0UspHMn7HGsHUESsVeV0fLqlfXyd2RbHTmDMP6w== Kubernikus Master Key",
			NodePools: []models.NodePool{
				{
					Name:   "small",
					Flavor: "m1.small",
					Image:  "coreos-stable-amd64",
					Size:   ClusterSmallNodePoolSize,
				},
			},
		},
	}
}

func mediumNodePoolItem() *models.NodePool {
	return &models.NodePool{
		Name:   "medium",
		Image:  "coreos-stable-amd64",
		Flavor: "m1.medium",
		Size:   ClusterMediumNodePoolSize,
	}
}

func newE2ESmokeTestCluster(klusterName string) *models.Kluster {
	return &models.Kluster{
		Name: klusterName,
		Spec: models.KlusterSpec{
			NodePools: []models.NodePool{
				{
					Name:   "small",
					Flavor: "m1.small",
					Image:  "coreos-stable-amd64",
					Size:   ClusterSmallNodePoolSize,
				},
				{
					Name:   "medium",
					Image:  "coreos-stable-amd64",
					Flavor: "m1.medium",
					Size:   ClusterMediumNodePoolSize,
				},
			},
		},
	}
}

func isPodRunning(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Deleted:
		return false, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
	}
	switch t := event.Object.(type) {
	case *v1.Pod:
		switch t.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, fmt.Errorf("pod failed or ran to completion")
		}
	}
	return false, nil
}

func (s *E2ETestSuite) waitForPodDeleted(namespace, name string) (bool, error) {
	for start := time.Now(); time.Since(start) < TimeoutPod; time.Sleep(CheckInterval) {
		select {
		default:
			_, err := s.clientSet.CoreV1().Pods(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					log.Printf("pod %v/%v was deleted", namespace, name)
					return true, nil
				}
				return false, err
			}
			log.Printf("pod %v/%v still exists", namespace, name)
		case <-s.stopCh:
			os.Exit(1)
		}
	}
	return false, nil
}

func (s *E2ETestSuite) handleError(err error) {
	if err == nil {
		return
	}
	log.Print(err)
	// cleanup
	//if !s.IsNoTeardown {
	//  s.tearDownCluster()
	//}
	os.Exit(1)
}

func (s *E2ETestSuite) tearDownCluster() {
	s.emptyNodePoolsOfKluster()
	log.Printf("Deleting cluster %v", s.ClusterName)

	_, err := s.kubernikusClient.Operations.TerminateCluster(
		operations.NewTerminateClusterParams().WithName(s.ClusterName),
		s.authFunc(),
	)
	s.handleError(err)

	if err := s.waitForCluster(
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
					log.Println("Failed to show cluster %v: %v", s.ClusterName, err)
					return false
				}
			}
			log.Printf("Cluster %v was not terminated. Still in %v", s.ClusterName, k.Status.Phase)
			return false
		},
	); err != nil {
		s.handleError(fmt.Errorf("error while waiting for cluster %v to be terminated: %v", s.ClusterName, err))
	}
}

func (s *E2ETestSuite) NewClientSet() {
	config := s.newClientConfig()
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Couldn't create Kubernetes client: %s", err)
		return
	}
	s.clientSet = clientSet
}

func (s *E2ETestSuite) newClientConfig() *rest.Config {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	if s.KubeConfig != "" {
		rules.ExplicitPath = s.KubeConfig
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		fmt.Errorf("couldn't get Kubernetes default config: %s", err)
	}

	return config
}

func (s *E2ETestSuite) isClusterUpOrWait() {
	err := s.waitForCluster(
		s.ClusterName,
		fmt.Sprintf("Cluster %s wasn't created in time", s.ClusterName),
		func(k *models.Kluster, err error) bool {
			if err != nil {
				switch err.(type) {
				case *operations.ShowClusterDefault:
					result := err.(*operations.ShowClusterDefault)
					if result.Code() == 404 {
						log.Printf("Cluster %v does not exist. Creating it", s.ClusterName)
						_, err := s.kubernikusClient.Operations.CreateCluster(
							operations.NewCreateClusterParams().WithBody(newE2ESmokeTestCluster(ClusterName)),
							s.authFunc(),
						)
						s.handleError(err)
						return true
					}
				case error:
					log.Printf("Failed to show cluster %v: %v", s.ClusterName, err)
					return false
				}
				return false
			}
			if k.Status.Phase == models.KlusterPhaseRunning {
				log.Printf("Cluster %v is ready. Checking nodes..", k.Name)
				if isNodesHealthyAndRunning(k.Status.NodePools, k.Spec.NodePools) {
					return true
				}
			} else {
				log.Printf("Cluster %v not ready for smoke test. Still in %v", k.Name, k.Status.Phase)
			}
			return false
		},
	)
	s.handleError(err)
}

func (s *E2ETestSuite) waitForPVC(pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	w, err := s.clientSet.CoreV1().PersistentVolumeClaims(pvc.GetNamespace()).Watch(
		meta_v1.SingleObject(
			meta_v1.ObjectMeta{
				Name: pvc.GetName(),
			},
		),
	)
	if err != nil {
		return nil, err
	}

	_, err = watch.Until(5*time.Minute, w, isPVCBound)
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func isPVCBound(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Deleted:
		return false, errors.NewNotFound(schema.GroupResource{Resource: "pvc"}, "")
	}
	switch t := event.Object.(type) {
	case *v1.PersistentVolumeClaim:
		switch t.Status.Phase {
		case v1.ClaimBound:
			return true, nil
		case v1.ClaimPending:
			return false, nil
		case v1.ClaimLost:
			return false, fmt.Errorf("pvc is lost")
		}
	}
	return false, nil
}

func (s *E2ETestSuite) exitGraceful(sigs chan os.Signal) {
	sigs <- syscall.SIGTERM
}
