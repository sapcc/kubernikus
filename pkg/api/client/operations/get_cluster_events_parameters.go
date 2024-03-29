// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewGetClusterEventsParams creates a new GetClusterEventsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetClusterEventsParams() *GetClusterEventsParams {
	return &GetClusterEventsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetClusterEventsParamsWithTimeout creates a new GetClusterEventsParams object
// with the ability to set a timeout on a request.
func NewGetClusterEventsParamsWithTimeout(timeout time.Duration) *GetClusterEventsParams {
	return &GetClusterEventsParams{
		timeout: timeout,
	}
}

// NewGetClusterEventsParamsWithContext creates a new GetClusterEventsParams object
// with the ability to set a context for a request.
func NewGetClusterEventsParamsWithContext(ctx context.Context) *GetClusterEventsParams {
	return &GetClusterEventsParams{
		Context: ctx,
	}
}

// NewGetClusterEventsParamsWithHTTPClient creates a new GetClusterEventsParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetClusterEventsParamsWithHTTPClient(client *http.Client) *GetClusterEventsParams {
	return &GetClusterEventsParams{
		HTTPClient: client,
	}
}

/*
GetClusterEventsParams contains all the parameters to send to the API endpoint

	for the get cluster events operation.

	Typically these are written to a http.Request.
*/
type GetClusterEventsParams struct {

	// Name.
	Name string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get cluster events params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterEventsParams) WithDefaults() *GetClusterEventsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get cluster events params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterEventsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get cluster events params
func (o *GetClusterEventsParams) WithTimeout(timeout time.Duration) *GetClusterEventsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get cluster events params
func (o *GetClusterEventsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get cluster events params
func (o *GetClusterEventsParams) WithContext(ctx context.Context) *GetClusterEventsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get cluster events params
func (o *GetClusterEventsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get cluster events params
func (o *GetClusterEventsParams) WithHTTPClient(client *http.Client) *GetClusterEventsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get cluster events params
func (o *GetClusterEventsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithName adds the name to the get cluster events params
func (o *GetClusterEventsParams) WithName(name string) *GetClusterEventsParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the get cluster events params
func (o *GetClusterEventsParams) SetName(name string) {
	o.Name = name
}

// WriteToRequest writes these params to a swagger request
func (o *GetClusterEventsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param name
	if err := r.SetPathParam("name", o.Name); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
