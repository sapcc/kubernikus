// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// GetAuthCallbackHandlerFunc turns a function with the right signature into a get auth callback handler
type GetAuthCallbackHandlerFunc func(GetAuthCallbackParams) middleware.Responder

// Handle executing the request and returning a response
func (fn GetAuthCallbackHandlerFunc) Handle(params GetAuthCallbackParams) middleware.Responder {
	return fn(params)
}

// GetAuthCallbackHandler interface for that can handle valid get auth callback params
type GetAuthCallbackHandler interface {
	Handle(GetAuthCallbackParams) middleware.Responder
}

// NewGetAuthCallback creates a new http.Handler for the get auth callback operation
func NewGetAuthCallback(ctx *middleware.Context, handler GetAuthCallbackHandler) *GetAuthCallback {
	return &GetAuthCallback{Context: ctx, Handler: handler}
}

/*
	GetAuthCallback swagger:route GET /auth/callback getAuthCallback

callback for oauth result
*/
type GetAuthCallback struct {
	Context *middleware.Context
	Handler GetAuthCallbackHandler
}

func (o *GetAuthCallback) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetAuthCallbackParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
