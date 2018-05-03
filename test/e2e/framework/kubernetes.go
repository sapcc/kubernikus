package framework

import (
	"fmt"
	"time"

	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	PodListTimeout                 = 1 * time.Minute
	ServiceAccountProvisionTimeout = 2 * time.Minute
	Poll                           = 2 * time.Second
)

type Kubernetes struct {
	ClientSet        *kubernetes.Clientset
	restClientConfig *restclient.Config
}

func NewKubernetesFramework(kubernikus *Kubernikus, kluster string) (*Kubernetes, error) {
	credentials, err := kubernikus.Client.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(kluster),
		kubernikus.AuthInfo,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get Kubeconfig: %s", err)
	}

	apiConfig, err := clientcmd.Load([]byte(credentials.Payload.Kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("couldn't parse kubeconfig: %s", err)
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't create rest config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("couldn't produce clientset: %v", err)
	}

	return &Kubernetes{
		ClientSet:        clientset,
		restClientConfig: restConfig,
	}, nil
}
