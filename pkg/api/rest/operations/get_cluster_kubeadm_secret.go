// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

// GetClusterKubeadmSecretHandlerFunc turns a function with the right signature into a get cluster kubeadm secret handler
type GetClusterKubeadmSecretHandlerFunc func(GetClusterKubeadmSecretParams, *models.Principal) middleware.Responder

// Handle executing the request and returning a response
func (fn GetClusterKubeadmSecretHandlerFunc) Handle(params GetClusterKubeadmSecretParams, principal *models.Principal) middleware.Responder {
	return fn(params, principal)
}

// GetClusterKubeadmSecretHandler interface for that can handle valid get cluster kubeadm secret params
type GetClusterKubeadmSecretHandler interface {
	Handle(GetClusterKubeadmSecretParams, *models.Principal) middleware.Responder
}

// NewGetClusterKubeadmSecret creates a new http.Handler for the get cluster kubeadm secret operation
func NewGetClusterKubeadmSecret(ctx *middleware.Context, handler GetClusterKubeadmSecretHandler) *GetClusterKubeadmSecret {
	return &GetClusterKubeadmSecret{Context: ctx, Handler: handler}
}

/*
	GetClusterKubeadmSecret swagger:route GET /api/v1/clusters/{name}/kubeadmsecret getClusterKubeadmSecret

Get CA secret for kubeadm
*/
type GetClusterKubeadmSecret struct {
	Context *middleware.Context
	Handler GetClusterKubeadmSecretHandler
}

func (o *GetClusterKubeadmSecret) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetClusterKubeadmSecretParams()
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
