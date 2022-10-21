package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/stretchr/testify/require"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	clientutil "github.com/sapcc/kubernikus/pkg/client/kubernetes"
	"github.com/sapcc/kubernikus/pkg/util/generator"
	"github.com/sapcc/kubernikus/test/e2e/framework"
)

var (
	kubernikusURL = flag.String("kubernikus", "", "Kubernikus URL")
	kluster       = flag.String("kluster", "", "Use existing Kluster")
	reuse         = flag.Bool("reuse", false, "Reuse exisiting Kluster")
	cleanup       = flag.Bool("cleanup", true, "Cleanup after tests have been run")
	isolate       = flag.Bool("isolate", false, "Do not destroy or depend on resources of other tests running in the same project")
)

const (
	SmokeTestNodeCount = 2
)

func validate() error {
	if *kubernikusURL == "" {
		return fmt.Errorf("You need to provide the --kubernikus flag")
	}

	k, err := url.Parse(*kubernikusURL)
	if err != nil || k.Host == "" {
		return fmt.Errorf("You need to provide an URL for --kubernikus: %v", err)
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

	klog.InitFlags(nil)
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

	if os.Getenv("ISOLATE_TEST") == "true" {
		*isolate = true
	}

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
		if os.Getenv("CP_OIDC_AUTH_URL") != "" {
			fmt.Printf("CP_OIDC_AUTH_URL:          %v\n", os.Getenv("CP_OIDC_AUTH_URL"))
			fmt.Printf("CP_OIDC_CONNECTOR_ID:      %v\n", os.Getenv("CP_OIDC_CONNECTOR_ID"))
			fmt.Printf("CP_OIDC_USERNAME:          %v\n", os.Getenv("CP_OIDC_USERNAME"))
		} else {
			fmt.Printf("CP_OS_PROJECT_NAME:        %v\n", os.Getenv("CP_OS_PROJECT_NAME"))
			fmt.Printf("CP_OS_PROJECT_DOMAIN_NAME: %v\n", os.Getenv("CP_OS_PROJECT_DOMAIN_NAME"))
		}
	}
	fmt.Printf("\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("Test Parameters\n")
	fmt.Printf("========================================================================\n")
	fmt.Printf("Kubernikus:                %v\n", kurl)
	fmt.Printf("Kluster Name:              %v\n", klusterName)
	fmt.Printf("Reuse:                     %v\n", *reuse)
	fmt.Printf("Cleanup:                   %v\n", *cleanup)
	fmt.Printf("Isolate:                   %v\n", *isolate)
	fmt.Println("")
	fmt.Printf("Dashboard:                 https://dashboard.%s.cloud.sap/%s/%s/kubernetes\n", os.Getenv("OS_REGION_NAME"), os.Getenv("OS_PROJECT_DOMAIN_NAME"), os.Getenv("OS_PROJECT_NAME"))
	if os.Getenv("CP_KLUSTER") != "" {
		fmt.Printf("CP Kluster Name:           %v\n", os.Getenv("CP_KLUSTER"))
	}
	if os.Getenv("KLUSTER_VERSION") != "" {
		fmt.Printf("Kubernetes Version:        %v\n", os.Getenv("KLUSTER_VERSION"))
	}
	if os.Getenv("KLUSTER_CIDR") != "" {
		fmt.Printf("Cluster CIDR:              %v\n", os.Getenv("KLUSTER_CIDR"))
	}
	if os.Getenv("KLUSTER_OS_IMAGES") != "" {
		fmt.Printf("OS Image(s):               %v\n", os.Getenv("KLUSTER_OS_IMAGES"))
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
	authInfo, err := framework.NewOpensStackAuth(authOptions)
	if err != nil {
		require.NoError(t, err, "Failed to create auth for kubernikus api")
	}
	kubernikus := framework.NewKubernikusFramework(kurl, authInfo)
	require.NoError(t, err, "Must be able to connect to Kubernikus")

	var kubernikusControlPlane *framework.Kubernikus
	if os.Getenv("CP_KUBERNIKUS_URL") != "" {
		kcpurl, err := url.Parse(os.Getenv("CP_KUBERNIKUS_URL"))
		require.NoError(t, err, "Must be able to parse Kubernikus control plane URL")

		var auth_info runtime.ClientAuthInfoWriter
		if os.Getenv("CP_OIDC_AUTH_URL") != "" {
			auth_info, err = framework.NewOIDCAuth(os.Getenv("CP_OIDC_USERNAME"), os.Getenv("CP_OIDC_PASSWORD"), os.Getenv("CP_OIDC_CONNECTOR_ID"), os.Getenv("CP_OIDC_AUTH_URL"))
			require.NoError(t, err, "Failed to use oidc auth for controlplane kubernikus")
		} else {
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
			auth_info, err = framework.NewOpensStackAuth(authOptionsControlPlane)
			require.NoError(t, err, "Failed to use openstack auth for controlplane kubernikus")
		}
		kubernikusControlPlane = framework.NewKubernikusFramework(kcpurl, auth_info)
	}

	openstack, err := framework.NewOpenStackFramework()
	require.NoError(t, err, "Must be able to connect to OpenStack")

	project, err := openstack.Provider.GetAuthResult().(tokens.CreateResult).ExtractProject()
	require.NoError(t, err, "Cannot extract project from token")
	fullKlusterName := fmt.Sprintf("%s-%s", klusterName, project.ID)

	// Pyrolize garbage left from previous e2e runs
	pyrolisisTests := &PyrolisisTests{kubernikus, openstack, *reuse, *isolate}
	if !t.Run("Pyrolisis", pyrolisisTests.Run) {
		return
	}

	preflightTests := &PreFlightTests{kubernikus, openstack, *reuse}
	if !t.Run("PreflightCheck", preflightTests.Run) {
		return
	}

	if cleanup != nil && *cleanup == true {
		cleanupTests := &CleanupTests{kubernikus, openstack, klusterName, *reuse, *isolate}
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

		var kubernetesControlPlane *framework.Kubernetes
		if os.Getenv("CP_KUBERNIKUS_URL") != "" {
			kubernetesControlPlane, err = framework.NewKubernetesFramework(kubernikusControlPlane, os.Getenv("CP_KLUSTER"))
			require.NoError(t, err, "Must be able to create a control plane kubernetes client")
		} else {
			if context := os.Getenv("CP_KLUSTER"); context != "" {
				c, err := clientutil.NewConfig("", context)
				require.NoErrorf(t, err, "Failed to get rest config for context %s", context)
				cs, err := client.NewForConfig(c)
				require.NoError(t, err, "Failed to get clientset for config")
				kubernetesControlPlane = &framework.Kubernetes{ClientSet: cs, RestConfig: c}
			}
		}

		if kubernetesControlPlane != nil {
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
