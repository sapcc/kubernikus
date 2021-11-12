package common

import (
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"

	kubernikus "github.com/sapcc/kubernikus/pkg/api/client"
	"github.com/sapcc/kubernikus/pkg/api/client/operations"
	"github.com/sapcc/kubernikus/pkg/api/models"
)

type KubernikusClient struct {
	token  string
	client *kubernikus.Kubernikus
}

func NewKubernikusClient(url *url.URL, token string) *KubernikusClient {
	transport := kubernikus.DefaultTransportConfig().
		WithSchemes([]string{url.Scheme}).
		WithHost(url.Host).
		WithBasePath(url.EscapedPath())

	return &KubernikusClient{
		token:  token,
		client: kubernikus.NewHTTPClientWithConfig(nil, transport),
	}
}

func (k *KubernikusClient) authFunc() runtime.ClientAuthInfoWriterFunc {
	return runtime.ClientAuthInfoWriterFunc(
		func(req runtime.ClientRequest, reg strfmt.Registry) error {
			req.SetHeaderParam("X-AUTH-TOKEN", k.token)
			return nil
		})
}

func (k *KubernikusClient) GetCredentials(name string) (string, error) {
	ok, err := k.client.Operations.GetClusterCredentials(
		operations.NewGetClusterCredentialsParams().WithName(name),
		k.authFunc())

	switch err.(type) {
	case *operations.GetClusterCredentialsDefault:
		result := err.(*operations.GetClusterCredentialsDefault)
		if result.Code() == 404 {
			return "", errors.Errorf("Cluster %v not found", name)
		}
		return "", errors.Errorf(result.Payload.Message)
	case error:
		return "", errors.Wrapf(err, "A generic error occurred")
	}

	return ok.Payload.Kubeconfig, nil
}

func (k *KubernikusClient) GetCredentialsOIDC(name string) (string, error) {
	ok, err := k.client.Operations.GetClusterCredentialsOIDC(
		operations.NewGetClusterCredentialsOIDCParams().WithName(name),
		k.authFunc())

	switch err.(type) {
	case *operations.GetClusterCredentialsOIDCDefault:
		result := err.(*operations.GetClusterCredentialsOIDCDefault)
		if result.Code() == 404 {
			return "", errors.Errorf("Cluster %v not found", name)
		}
		return "", errors.Errorf(result.Payload.Message)
	case error:
		return "", errors.Wrapf(err, "A generic error occurred")
	}

	return ok.Payload.Kubeconfig, nil
}

func (k *KubernikusClient) CreateCluster(cluster *models.Kluster) error {
	params := operations.NewCreateClusterParams().WithBody(cluster)
	_, err := k.client.Operations.CreateCluster(params, k.authFunc())
	switch err.(type) {
	case *operations.CreateClusterDefault:
		result := err.(*operations.CreateClusterDefault)
		return errors.Errorf(result.Payload.Message)
	case error:
		return errors.Wrap(err, "Error creating cluster")
	}
	return nil
}

func (k *KubernikusClient) DeleteCluster(name string) error {
	params := operations.NewTerminateClusterParams().WithName(name)
	_, err := k.client.Operations.TerminateCluster(params, k.authFunc())
	switch err.(type) {
	case *operations.TerminateClusterDefault:
		result := err.(*operations.TerminateClusterDefault)
		return errors.Errorf(result.Payload.Message)
	case error:
		return errors.Wrap(err, "Error deleting cluster")
	}
	return nil
}

func (k *KubernikusClient) ShowCluster(name string) (*models.Kluster, error) {
	params := operations.NewShowClusterParams()
	params.Name = name
	ok, err := k.client.Operations.ShowCluster(params, k.authFunc())
	switch err.(type) {
	case *operations.ShowClusterDefault:
		result := err.(*operations.ShowClusterDefault)
		return nil, errors.Errorf(result.Payload.Message)
	case error:
		return nil, errors.Wrap(err, "Getting cluster failed")
	}
	return ok.Payload, nil
}

func (k *KubernikusClient) GetClusterValues(account, name string) (string, error) {
	params := operations.NewGetClusterValuesParams()
	params.Name = name
	params.Account = account

	ok, err := k.client.Operations.GetClusterValues(params, k.authFunc())
	switch err.(type) {
	case *operations.GetClusterValuesDefault:
		result := err.(*operations.GetClusterValuesDefault)
		return "", errors.Errorf(result.Payload.Message)
	case error:
		return "", errors.Wrap(err, "Getting cluster failed")
	}
	return ok.Payload.Values, nil
}

func (k *KubernikusClient) ListAllClusters() ([]*models.Kluster, error) {
	ok, err := k.client.Operations.ListClusters(operations.NewListClustersParams(), k.authFunc())
	switch err.(type) {
	case *operations.ListClustersDefault:
		result := err.(*operations.ListClustersDefault)
		return nil, errors.Errorf(result.Payload.Message)
	case error:
		return nil, errors.Wrapf(err, "Listing clusters failed")
	}
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't fetch cluster list from Kubernikus API")
	}
	if len(ok.Payload) == 0 {
		return nil, errors.Errorf("There's no cluster in this project")
	}
	return ok.Payload, nil
}

func (k *KubernikusClient) ListNodePools(clusterName string) ([]models.NodePool, error) {
	ok, err := k.ShowCluster(clusterName)
	if err != nil {
		return nil, err
	}
	return ok.Spec.NodePools, nil
}

func (k *KubernikusClient) ShowNodePool(clusterName string, nodePoolName string) (*models.NodePool, error) {
	ok, err := k.ShowCluster(clusterName)
	if err != nil {
		return nil, err
	}
	for _, nodePool := range ok.Spec.NodePools {
		if nodePool.Name == nodePoolName {
			return &nodePool, nil
		}
	}
	return nil, nil
}

func (k *KubernikusClient) GetDefaultCluster() (*models.Kluster, error) {
	ok, err := k.client.Operations.ListClusters(operations.NewListClustersParams(), k.authFunc())

	switch err.(type) {
	case *operations.ListClustersDefault:
		result := err.(*operations.ListClustersDefault)
		return nil, errors.Errorf(result.Payload.Message)
	case error:
		return nil, errors.Wrapf(err, "Listing clusters failed")
	}

	if err != nil {
		return nil, errors.Wrap(err, "Couldn't fetch cluster list from Kubernikus API")
	}

	if len(ok.Payload) == 0 {
		return nil, errors.Errorf("There's no cluster in this project")
	}

	if len(ok.Payload) > 1 {
		return nil, errors.Errorf("There's more than one cluster in this project.")
	}

	return ok.Payload[0], nil
}
