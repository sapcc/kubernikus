// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

// GetClusterCredentialsOIDCHandlerFunc turns a function with the right signature into a get cluster credentials o ID c handler
type GetClusterCredentialsOIDCHandlerFunc func(GetClusterCredentialsOIDCParams, *models.Principal) middleware.Responder

// Handle executing the request and returning a response
func (fn GetClusterCredentialsOIDCHandlerFunc) Handle(params GetClusterCredentialsOIDCParams, principal *models.Principal) middleware.Responder {
	return fn(params, principal)
}

// GetClusterCredentialsOIDCHandler interface for that can handle valid get cluster credentials o ID c params
type GetClusterCredentialsOIDCHandler interface {
	Handle(GetClusterCredentialsOIDCParams, *models.Principal) middleware.Responder
}

// NewGetClusterCredentialsOIDC creates a new http.Handler for the get cluster credentials o ID c operation
func NewGetClusterCredentialsOIDC(ctx *middleware.Context, handler GetClusterCredentialsOIDCHandler) *GetClusterCredentialsOIDC {
	return &GetClusterCredentialsOIDC{Context: ctx, Handler: handler}
}

/*
	GetClusterCredentialsOIDC swagger:route GET /api/v1/clusters/{name}/credentials/oidc getClusterCredentialsOIdC

Get user specific credentials to access the cluster with OIDC
*/
type GetClusterCredentialsOIDC struct {
	Context *middleware.Context
	Handler GetClusterCredentialsOIDCHandler
}

func (o *GetClusterCredentialsOIDC) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetClusterCredentialsOIDCParams()
	uprinc, aCtx, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	if aCtx != nil {
		*r = *aCtx
	}
	var principal *models.Principal
	if uprinc != nil {
		principal = uprinc.(*models.Principal) // this is really a models.Principal, I promise
	}

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params, principal) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
