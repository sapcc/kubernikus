// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	middleware "github.com/go-openapi/runtime/middleware"

	models "github.com/sapcc/kubernikus/pkg/api/models"
)

// GetBoostrapConfigHandlerFunc turns a function with the right signature into a get boostrap config handler
type GetBoostrapConfigHandlerFunc func(GetBoostrapConfigParams, *models.Principal) middleware.Responder

// Handle executing the request and returning a response
func (fn GetBoostrapConfigHandlerFunc) Handle(params GetBoostrapConfigParams, principal *models.Principal) middleware.Responder {
	return fn(params, principal)
}

// GetBoostrapConfigHandler interface for that can handle valid get boostrap config params
type GetBoostrapConfigHandler interface {
	Handle(GetBoostrapConfigParams, *models.Principal) middleware.Responder
}

// NewGetBoostrapConfig creates a new http.Handler for the get boostrap config operation
func NewGetBoostrapConfig(ctx *middleware.Context, handler GetBoostrapConfigHandler) *GetBoostrapConfig {
	return &GetBoostrapConfig{Context: ctx, Handler: handler}
}

/*GetBoostrapConfig swagger:route GET /api/v1/clusters/{name}/bootstrap getBoostrapConfig

Get bootstrap config to onboard a node

*/
type GetBoostrapConfig struct {
	Context *middleware.Context
	Handler GetBoostrapConfigHandler
}

func (o *GetBoostrapConfig) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		r = rCtx
	}
	var Params = NewGetBoostrapConfigParams()

	uprinc, aCtx, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	if aCtx != nil {
		r = aCtx
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
