package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/kubernikus/pkg/util/generator"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

var (
	kubernikusURL = flag.String("kubernikus", "", "Kubernikus URL")
	kluster       = flag.String("kluster", "", "Use existing Kluster")
	reuse         = flag.Bool("reuse", false, "Reuse exisiting Kluster")
	cleanup       = flag.Bool("cleanup", true, "Cleanup after tests have been run")
)

const (
	SmokeTestNodeCount = 2
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
	klusterName := generator.SimpleNameGenerator.GenerateName("e2e-")

	if kluster != nil && *kluster != "" {
		klusterName = *kluster
	}

	kurl, err := url.Parse(*kubernikusURL)
	require.NoError(t, err, "Must be able to parse Kubernikus URL")
	require.NotEmpty(t, kurl.Host, "There must be a host in the Kubernikus URL")

	fmt.Printf("========================================================================\n")
	fmt.Printf("Authentication\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("OS_AUTH_URL:               %v\n", os.Getenv("OS_AUTH_URL"))
	fmt.Printf("OS_USERNAME:               %v\n", os.Getenv("OS_USERNAME"))
	fmt.Printf("OS_USER_DOMAIN_NAME:       %v\n", os.Getenv("OS_USER_DOMAIN_NAME"))
	fmt.Printf("OS_PROJECT_NAME:           %v\n", os.Getenv("OS_PROJECT_NAME"))
	fmt.Printf("OS_PROJECT_DOMAIN_NAME:    %v\n", os.Getenv("OS_PROJECT_DOMAIN_NAME"))
	fmt.Printf("\n")
	if os.Getenv("CP_KUBERNIKUS_URL") != "" {
		fmt.Printf("CP_KUBERNIKUS_URL:         %v\n", os.Getenv("CP_KUBERNIKUS_URL"))
		fmt.Printf("CP_OS_PROJECT_NAME:        %v\n", os.Getenv("CP_OS_PROJECT_NAME"))
		fmt.Printf("CP_OS_PROJECT_DOMAIN_NAME: %v\n", os.Getenv("CP_OS_PROJECT_DOMAIN_NAME"))
	}
	fmt.Printf("\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("Test Parameters\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("Kubernikus:                %v\n", kurl.Host)
	fmt.Printf("Kluster Name:              %v\n", klusterName)
	fmt.Printf("Reuse:                     %v\n", *reuse)
	fmt.Printf("Cleanup:                   %v\n", *cleanup)
	fmt.Println("")
	fmt.Printf("Dashboard:                 https://dashboard.%s.cloud.sap/%s/%s/kubernetes\n", os.Getenv("OS_REGION_NAME"), os.Getenv("OS_PROJECT_DOMAIN_NAME"), os.Getenv("OS_PROJECT_NAME"))
	if os.Getenv("CP_KUBERNIKUS_URL") != "" {
		fmt.Printf("CP Kluster Name:           %v\n", os.Getenv("CP_KLUSTER"))
	}
	fmt.Printf("\n\n")

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: os.Getenv("OS_PROJECT_NAME"),
			DomainName:  os.Getenv("OS_PROJECT_DOMAIN_NAME"),
		},
	}
	kubernikus, err := framework.NewKubernikusFramework(kurl, authOptions)
	require.NoError(t, err, "Must be able to connect to Kubernikus")

	var kubernikusControlPlane *framework.Kubernikus
	if os.Getenv("CP_KUBERNIKUS_URL") != "" {
		kcpurl, err := url.Parse(os.Getenv("CP_KUBERNIKUS_URL"))
		require.NoError(t, err, "Must be able to parse Kubernikus control plane URL")
		authOptionsControlPlane := &tokens.AuthOptions{
			IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
			Username:         os.Getenv("OS_USERNAME"),
			Password:         os.Getenv("OS_PASSWORD"),
			DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
			AllowReauth:      true,
			Scope: tokens.Scope{
				ProjectName: os.Getenv("CP_OS_PROJECT_NAME"),
				DomainName:  os.Getenv("CP_OS_PROJECT_DOMAIN_NAME"),
			},
		}
		kubernikusControlPlane, err = framework.NewKubernikusFramework(kcpurl, authOptionsControlPlane)
		require.NoError(t, err, "Must be able to connect to Kubernikus Control Plane")
	}

	openstack, err := framework.NewOpenStackFramework()
	require.NoError(t, err, "Must be able to connect to OpenStack")

	project, err := openstack.Provider.GetAuthResult().(tokens.CreateResult).ExtractProject()
	require.NoError(t, err, "Cannot extract project from token")
	fullKlusterName := fmt.Sprintf("%s-%s", klusterName, project.ID)

	// Pyrolize garbage left from previous e2e runs
	pyrolisisTests := &PyrolisisTests{kubernikus, openstack, *reuse}
	if !t.Run("Pyrolisis", pyrolisisTests.Run) {
		return
	}

	preflightTests := &PreFlightTests{kubernikus, openstack, *reuse}
	if !t.Run("PreflightCheck", preflightTests.Run) {
		return
	}

	if cleanup != nil && *cleanup == true {
		cleanupTests := &CleanupTests{kubernikus, openstack, klusterName, *reuse}
		defer t.Run("Cleanup", cleanupTests.Run)
	}

	setupTests := &SetupTests{kubernikus, openstack, klusterName, *reuse}
	if !t.Run("Setup", setupTests.Run) {
		return
	}

	kubernetes, err := framework.NewKubernetesFramework(kubernikus, klusterName)
	require.NoError(t, err, "Must be able to create a kubernetes client")

	apiTests := &APITests{kubernikus, klusterName}
	t.Run("API", apiTests.Run)

	nodeTests := &NodeTests{kubernetes, kubernikus, openstack, SmokeTestNodeCount, klusterName}
	if !t.Run("Nodes", nodeTests.Run) {
		return
	}

	t.Run("Smoke", func(t *testing.T) {
		volumeTests := &VolumeTests{Kubernetes: kubernetes}
		t.Run("Volumes", volumeTests.Run)

		networkTests := &NetworkTests{Kubernetes: kubernetes}
		t.Run("Network", networkTests.Run)

		if os.Getenv("CP_KUBERNIKUS_URL") != "" {
			kubernetesControlPlane, err := framework.NewKubernetesFramework(kubernikusControlPlane, os.Getenv("CP_KLUSTER"))
			require.NoError(t, err, "Must be able to create a control plane kubernetes client")

			namespace := "kubernikus"
			if os.Getenv("CP_NAMESPACE") != "" {
				namespace = os.Getenv("CP_NAMESPACE")
			}

			etcdBackupTests := &EtcdBackupTests{
				KubernetesControlPlane: kubernetesControlPlane,
				Kubernetes:             kubernetes,
				FullKlusterName:        fullKlusterName,
				Namespace:              namespace,
			}
			t.Run("EtcdBackupTests", etcdBackupTests.Run)
		}
	})
}

func runParallel(t *testing.T) {
	if os.Getenv("RUN_PARALLEL") != "false" {
		t.Parallel()
	}
}
