// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

// GetClusterInfoHandlerFunc turns a function with the right signature into a get cluster info handler
type GetClusterInfoHandlerFunc func(GetClusterInfoParams, *models.Principal) middleware.Responder

// Handle executing the request and returning a response
func (fn GetClusterInfoHandlerFunc) Handle(params GetClusterInfoParams, principal *models.Principal) middleware.Responder {
	return fn(params, principal)
}

// GetClusterInfoHandler interface for that can handle valid get cluster info params
type GetClusterInfoHandler interface {
	Handle(GetClusterInfoParams, *models.Principal) middleware.Responder
}

// NewGetClusterInfo creates a new http.Handler for the get cluster info operation
func NewGetClusterInfo(ctx *middleware.Context, handler GetClusterInfoHandler) *GetClusterInfo {
	return &GetClusterInfo{Context: ctx, Handler: handler}
}

/*
	GetClusterInfo swagger:route GET /api/v1/clusters/{name}/info getClusterInfo

Get user specific info about the cluster
*/
type GetClusterInfo struct {
	Context *middleware.Context
	Handler GetClusterInfoHandler
}

func (o *GetClusterInfo) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetClusterInfoParams()
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
