package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/util"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

var (
	kubernikusURL = flag.String("kubernikus", "", "Kubernikus URL")
	kluster       = flag.String("kluster", "", "Use existing Kluster")
	reuse         = flag.Bool("reuse", false, "Reuse exisiting Kluster")
	cleanup       = flag.Bool("cleanup", true, "Cleanup after tests have been run")
)

func validate() error {
	if *kubernikusURL == "" {
		return fmt.Errorf("You need to provide the --kubernikus flag")
	}

	k, err := url.Parse(*kubernikusURL)
	if err != nil {
		return fmt.Errorf("You need to provide an URL for --kubernikus: %v", err)
	}

	if k.Host == "" {
		return fmt.Errorf("You need to provide an URL for --kubernikus")
	}

	if reuse != nil && *reuse && (kluster == nil || *kluster == "") {
		return fmt.Errorf("You need to provide the --kluster flag when --reuse is active")
	}

	for _, env := range []string{"OS_AUTH_URL", "OS_USERNAME", "OS_PASSWORD",
		"OS_USER_DOMAIN_NAME", "OS_PROJECT_NAME", "OS_PROJECT_DOMAIN_NAME"} {
		if os.Getenv(env) == "" {
			return fmt.Errorf("You need to provide %s", env)
		}
	}

	return nil
}

func TestMain(m *testing.M) {
	flag.Parse()

	if err := validate(); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	os.Exit(m.Run())
}

func TestRunner(t *testing.T) {
	namespaceNetwork := util.SimpleNameGenerator.GenerateName("e2e-network-")
	namespaceVolumes := util.SimpleNameGenerator.GenerateName("e2e-volumes-")
	klusterName := util.SimpleNameGenerator.GenerateName("e2e-")

	if kluster != nil && *kluster != "" {
		klusterName = *kluster
	}

	kurl, err := url.Parse(*kubernikusURL)
	require.NoError(t, err, "Must be able to parse Kubernikus URL")
	require.NotEmpty(t, kurl.Host, "There must be a host in the Kubernikus URL")

	fmt.Printf("========================================================================\n")
	fmt.Printf("Authentication\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("OS_AUTH_URL:            %v\n", os.Getenv("OS_AUTH_URL"))
	fmt.Printf("OS_USERNAME:            %v\n", os.Getenv("OS_USERNAME"))
	fmt.Printf("OS_USER_DOMAIN_NAME:    %v\n", os.Getenv("OS_USER_DOMAIN_NAME"))
	fmt.Printf("OS_PROJECT_NAME:        %v\n", os.Getenv("OS_PROJECT_NAME"))
	fmt.Printf("OS_PROJECT_DOMAIN_NAME: %v\n", os.Getenv("OS_PROJECT_DOMAIN_NAME"))
	fmt.Printf("\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("Test Parameters\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("Kubernikus:             %v\n", kurl.Host)
	fmt.Printf("Kluster Name:           %v\n", klusterName)
	fmt.Printf("Reuse:                  %v\n", *reuse)
	fmt.Printf("Cleanup:                %v\n", *cleanup)
	fmt.Printf("\n\n")

	kubernikus, err := framework.NewKubernikusFramework(kurl)
	require.NoError(t, err, "Must be able to connect to Kubernikus")

	api := APITests{kubernikus, klusterName}
	kluster := KlusterTests{kubernikus, klusterName}

	if cleanup != nil && *cleanup == true {
		defer t.Run("Cleanup", func(t *testing.T) {
			if t.Run("TerminateCluster", api.TerminateCluster) {
				t.Run("BecomesTerminating", kluster.KlusterPhaseBecomesTerminating)
				t.Run("IsDeleted", api.WaitForKlusterToBeDeleted)
			}
		})
	}

	setup := t.Run("Setup", func(t *testing.T) {
		if reuse == nil || *reuse == false {
			created := t.Run("CreateCluster", api.CreateCluster)
			require.True(t, created, "The Kluster must have been created")

			t.Run("BecomesCreating", kluster.KlusterPhaseBecomesCreating)
		}

		running := t.Run("BecomesRunning", kluster.KlusterPhaseBecomesRunning)
		require.True(t, running, "The Kluster must be Running")

		ready := t.Run("NodesBecomeReady", api.WaitForNodesReady)
		require.True(t, ready, "The Kluster must have Ready nodes")
	})
	require.True(t, setup, "Test setup must complete successfully")

	t.Run("API", func(t *testing.T) {
		t.Run("ListCluster", api.ListClusters)
		t.Run("ShowCluster", api.ShowCluster)
		t.Run("GetClusterInfo", api.GetClusterInfo)
		t.Run("GetCredentials", api.GetCredentials)
	})

	kubernetes, err := framework.NewKubernetesFramework(kubernikus, klusterName)
	require.NoError(t, err, "Must be able to create a kubernetes client")

	nodes, err := kubernetes.ClientSet.CoreV1().Nodes().List(meta_v1.ListOptions{})
	require.NoError(t, err, "There must be no error while listing the kluster's nodes")
	require.NotEqual(t, len(nodes.Items), 0, "There must be at least 2 nodes")
	require.NotEqual(t, len(nodes.Items), 1, "There must be at least 2 nodes")

	t.Run("Smoke", func(t *testing.T) {
		t.Run("Network", func(t *testing.T) {
			t.Parallel()
			network := NetworkTests{kubernetes, nodes, namespaceNetwork}

			defer t.Run("Cleanup", network.DeleteNamespace)
			t.Run("Setup", func(t *testing.T) {
				t.Run("Namespace/Create", network.CreateNamespace)
				t.Run("Namespace/Wait", network.WaitForNamespace)
				t.Run("Pods", func(t *testing.T) {
					t.Parallel()
					t.Run("Create", network.CreatePods)
					t.Run("Wait", network.WaitForPodsRunning)
				})
				t.Run("Services", func(t *testing.T) {
					t.Parallel()
					t.Run("Create", network.CreateServices)
					t.Run("Wait", network.WaitForServiceEndpoints)
				})
			})

			t.Run("Connectivity/Pods", network.TestPods)
			t.Run("Connectivity/Services", network.TestServices)
			t.Run("ConnectivityServicesWithDNS", network.TestServicesWithDNS)
		})

		t.Run("Volumes", func(t *testing.T) {
			t.Parallel()
			volumes := VolumeTests{kubernetes, nodes, nil, namespaceVolumes}

			defer t.Run("Cleanup", volumes.DeleteNamespace)
			t.Run("Setup/Namespace", func(t *testing.T) {
				t.Run("Create", volumes.CreateNamespace)
				t.Run("Wait", volumes.WaitForNamespace)
			})
			t.Run("PVC", func(t *testing.T) {
				t.Run("Create", volumes.CreatePVC)
				t.Run("Wait", volumes.WaitForPVCBound)
			})
			t.Run("Pods", func(t *testing.T) {
				t.Run("Create", volumes.CreatePod)
				t.Run("Wait", volumes.WaitForPVCPodsRunning)
			})
		})
	})
}
