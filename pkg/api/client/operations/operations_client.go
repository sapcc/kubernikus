// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new operations API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for operations API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	CreateCluster(params *CreateClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateClusterCreated, error)

	GetAuthCallback(params *GetAuthCallbackParams, opts ...ClientOption) (*GetAuthCallbackOK, error)

	GetAuthLogin(params *GetAuthLoginParams, opts ...ClientOption) error

	GetBootstrapConfig(params *GetBootstrapConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetBootstrapConfigOK, error)

	GetClusterCredentials(params *GetClusterCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterCredentialsOK, error)

	GetClusterCredentialsOIDC(params *GetClusterCredentialsOIDCParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterCredentialsOIDCOK, error)

	GetClusterEvents(params *GetClusterEventsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterEventsOK, error)

	GetClusterInfo(params *GetClusterInfoParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterInfoOK, error)

	GetClusterValues(params *GetClusterValuesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterValuesOK, error)

	GetClusters(params *GetClustersParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClustersOK, error)

	GetOpenstackMetadata(params *GetOpenstackMetadataParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetOpenstackMetadataOK, error)

	Info(params *InfoParams, opts ...ClientOption) (*InfoOK, error)

	ListAPIVersions(params *ListAPIVersionsParams, opts ...ClientOption) (*ListAPIVersionsOK, error)

	ListClusters(params *ListClustersParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListClustersOK, error)

	ShowCluster(params *ShowClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ShowClusterOK, error)

	TerminateCluster(params *TerminateClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*TerminateClusterAccepted, error)

	UpdateCluster(params *UpdateClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*UpdateClusterOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
CreateCluster creates a cluster
*/
func (a *Client) CreateCluster(params *CreateClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateClusterCreated, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateClusterParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "CreateCluster",
		Method:             "POST",
		PathPattern:        "/api/v1/clusters",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &CreateClusterReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*CreateClusterCreated)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*CreateClusterDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetAuthCallback callbacks for oauth result
*/
func (a *Client) GetAuthCallback(params *GetAuthCallbackParams, opts ...ClientOption) (*GetAuthCallbackOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetAuthCallbackParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetAuthCallback",
		Method:             "GET",
		PathPattern:        "/auth/callback",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetAuthCallbackReader{formats: a.formats},
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetAuthCallbackOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetAuthCallbackDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetAuthLogin logins through oauth2 server
*/
func (a *Client) GetAuthLogin(params *GetAuthLoginParams, opts ...ClientOption) error {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetAuthLoginParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetAuthLogin",
		Method:             "GET",
		PathPattern:        "/auth/login",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetAuthLoginReader{formats: a.formats},
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	_, err := a.transport.Submit(op)
	if err != nil {
		return err
	}
	return nil
}

/*
GetBootstrapConfig gets bootstrap config to onboard a node
*/
func (a *Client) GetBootstrapConfig(params *GetBootstrapConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetBootstrapConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetBootstrapConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetBootstrapConfig",
		Method:             "GET",
		PathPattern:        "/api/v1/clusters/{name}/bootstrap",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetBootstrapConfigReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetBootstrapConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetBootstrapConfigDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetClusterCredentials gets user specific credentials to access the cluster
*/
func (a *Client) GetClusterCredentials(params *GetClusterCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetClusterCredentialsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetClusterCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/clusters/{name}/credentials",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetClusterCredentialsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetClusterCredentialsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetClusterCredentialsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetClusterCredentialsOIDC gets user specific credentials to access the cluster with o ID c
*/
func (a *Client) GetClusterCredentialsOIDC(params *GetClusterCredentialsOIDCParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterCredentialsOIDCOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetClusterCredentialsOIDCParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetClusterCredentialsOIDC",
		Method:             "GET",
		PathPattern:        "/api/v1/clusters/{name}/credentials/oidc",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetClusterCredentialsOIDCReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetClusterCredentialsOIDCOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetClusterCredentialsOIDCDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetClusterEvents gets recent events about the cluster
*/
func (a *Client) GetClusterEvents(params *GetClusterEventsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterEventsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetClusterEventsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetClusterEvents",
		Method:             "GET",
		PathPattern:        "/api/v1/clusters/{name}/events",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetClusterEventsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetClusterEventsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetClusterEventsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetClusterInfo gets user specific info about the cluster
*/
func (a *Client) GetClusterInfo(params *GetClusterInfoParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterInfoOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetClusterInfoParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetClusterInfo",
		Method:             "GET",
		PathPattern:        "/api/v1/clusters/{name}/info",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetClusterInfoReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetClusterInfoOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetClusterInfoDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetClusterValues gets values for cluster chart admin only
*/
func (a *Client) GetClusterValues(params *GetClusterValuesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClusterValuesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetClusterValuesParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetClusterValues",
		Method:             "GET",
		PathPattern:        "/api/v1/{account}/clusters/{name}/values",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetClusterValuesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetClusterValuesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetClusterValuesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetClusters gets all clusters in a project admin only
*/
func (a *Client) GetClusters(params *GetClustersParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetClustersOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetClustersParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetClusters",
		Method:             "GET",
		PathPattern:        "/api/v1/{account}/clusters",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetClustersReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetClustersOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetClustersDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetOpenstackMetadata grabs bag of openstack metadata
*/
func (a *Client) GetOpenstackMetadata(params *GetOpenstackMetadataParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetOpenstackMetadataOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetOpenstackMetadataParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "GetOpenstackMetadata",
		Method:             "GET",
		PathPattern:        "/api/v1/openstack/metadata",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetOpenstackMetadataReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetOpenstackMetadataOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetOpenstackMetadataDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
Info gets info about kubernikus
*/
func (a *Client) Info(params *InfoParams, opts ...ClientOption) (*InfoOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewInfoParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "Info",
		Method:             "GET",
		PathPattern:        "/info",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &InfoReader{formats: a.formats},
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*InfoOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for Info: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
ListAPIVersions lists available api versions
*/
func (a *Client) ListAPIVersions(params *ListAPIVersionsParams, opts ...ClientOption) (*ListAPIVersionsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAPIVersionsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "ListAPIVersions",
		Method:             "GET",
		PathPattern:        "/api",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAPIVersionsReader{formats: a.formats},
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListAPIVersionsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for ListAPIVersions: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
ListClusters lists available clusters
*/
func (a *Client) ListClusters(params *ListClustersParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListClustersOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListClustersParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "ListClusters",
		Method:             "GET",
		PathPattern:        "/api/v1/clusters",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListClustersReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListClustersOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListClustersDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ShowCluster shows the specified cluster
*/
func (a *Client) ShowCluster(params *ShowClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ShowClusterOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewShowClusterParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "ShowCluster",
		Method:             "GET",
		PathPattern:        "/api/v1/clusters/{name}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ShowClusterReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ShowClusterOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ShowClusterDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
TerminateCluster terminates the specified cluster
*/
func (a *Client) TerminateCluster(params *TerminateClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*TerminateClusterAccepted, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewTerminateClusterParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "TerminateCluster",
		Method:             "DELETE",
		PathPattern:        "/api/v1/clusters/{name}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &TerminateClusterReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*TerminateClusterAccepted)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*TerminateClusterDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
UpdateCluster updates the specified cluster
*/
func (a *Client) UpdateCluster(params *UpdateClusterParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*UpdateClusterOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewUpdateClusterParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "UpdateCluster",
		Method:             "PUT",
		PathPattern:        "/api/v1/clusters/{name}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &UpdateClusterReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*UpdateClusterOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*UpdateClusterDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
