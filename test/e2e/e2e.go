package main

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"

	kubernikusClient "github.com/sapcc/kubernikus/pkg/api/client"

	"k8s.io/client-go/kubernetes"
)

type E2ETestSuite struct {
	E2ETestSuiteOptions
	testing *testing.T

	clientSet *kubernetes.Clientset

	ClusterName string

	kubernikusClient *kubernikusClient.Kubernikus

	timeout       int
	readyNodes    []v1.Node
	readyPods     []v1.Pod
	readyServices []v1.Service
	kubeConfig    string

	stopCh chan bool
	sigCh  chan os.Signal
}

func NewE2ETestSuite(t *testing.T, options E2ETestSuiteOptions) *E2ETestSuite {
	if err := options.OptionsFromConfigFile(); err != nil {
		glog.Fatal(err)
	}

	if err := options.Verify(); err != nil {
		log.Printf("Couldn't obtain openstack token using parameters given in config. Trying parameters from ENV. ")
		options.OpenStackCredentials = getOpenStackCredentialsFromENV()
		if err := options.Verify(); err != nil {
			glog.Errorf("Checked config and env. Insufficient parameters for authentication : %v", err)
			os.Exit(1)
		}
	}

	token, err := getToken(options.OpenStackCredentials)
	if err != nil {
		glog.Fatal("Authentication failed. Verify config file or environment")
	}

	options.OpenStackCredentials.Token = token

	kubernikusCli := kubernikusClient.NewHTTPClientWithConfig(
		nil,
		&kubernikusClient.TransportConfig{
			Host:    options.APIURL,
			Schemes: []string{"https"},
		},
	)

	return &E2ETestSuite{
		E2ETestSuiteOptions: options,
		testing:             t,
		ClusterName:         "e2e",
		timeout:             5,
		kubernikusClient:    kubernikusCli,
	}
}

func (s *E2ETestSuite) Run(wg *sync.WaitGroup, sigs chan os.Signal, stopCh chan bool) {
	defer wg.Done()
	wg.Add(1)

	s.stopCh = stopCh
	s.sigCh = sigs

	log.Println("Running tests")

	// API tests
	if s.IsTestCreate || s.IsTestAPI || s.IsTestAll {
		s.TestCreateCluster()
	}

	if s.IsTestAPI || s.IsTestAll {
		s.TestListClusters()
		s.TestShowCluster()
		s.TestUpdateCluster()
		s.TestGetClusterInfo()
	}

	// Smoke tests
	if s.IsTestSmoke || s.IsTestNetwork || s.IsTestVolume || s.IsTestAll {
		s.SetupSmokeTest()
		s.RunSmokeTest()
	}

	if s.IsTestDelete || s.IsTestAPI || s.IsTestAll {
		s.TestTerminateCluster()
	}

	log.Println("[passed all tests]")

	//stopCh <- true
	s.exitGraceful(sigs)
}
