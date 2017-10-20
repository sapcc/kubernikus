package common

import (
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"

	kubernikus "github.com/sapcc/kubernikus/pkg/client/kubernikus_generated"
	"github.com/sapcc/kubernikus/pkg/client/kubernikus_generated/operations"
	"github.com/sapcc/kubernikus/pkg/client/models"
)

type KubernikusClient struct {
	token  string
	client *kubernikus.Kubernikus
}

func NewKubernikusClient(url *url.URL, token string) *KubernikusClient {
	transport := kubernikus.DefaultTransportConfig().
		WithSchemes([]string{url.Scheme}).
		WithHost(url.Hostname()).
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
		return "", errors.Errorf(*result.Payload.Message)
	case error:
		return "", errors.Wrapf(err, "A generic error occured")
	}

	return ok.Payload.Kubeconfig, nil
}

func (k *KubernikusClient) GetDefaultCluster() (*models.Cluster, error) {
	ok, err := k.client.Operations.ListClusters(operations.NewListClustersParams(), k.authFunc())

	switch err.(type) {
	case *operations.ListClustersDefault:
		result := err.(*operations.ListClustersDefault)
		return nil, errors.Errorf(*result.Payload.Message)
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
