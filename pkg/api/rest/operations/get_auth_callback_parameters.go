// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/validate"

	strfmt "github.com/go-openapi/strfmt"
)

// NewGetAuthCallbackParams creates a new GetAuthCallbackParams object
// no default values defined in spec.
func NewGetAuthCallbackParams() GetAuthCallbackParams {

	return GetAuthCallbackParams{}
}

// GetAuthCallbackParams contains all the bound params for the get auth callback operation
// typically these are obtained from a http.Request
//
// swagger:parameters GetAuthCallback
type GetAuthCallbackParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  Required: true
	  In: query
	*/
	Code string
	/*
	  Required: true
	  In: query
	*/
	State string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGetAuthCallbackParams() beforehand.
func (o *GetAuthCallbackParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qCode, qhkCode, _ := qs.GetOK("code")
	if err := o.bindCode(qCode, qhkCode, route.Formats); err != nil {
		res = append(res, err)
	}

	qState, qhkState, _ := qs.GetOK("state")
	if err := o.bindState(qState, qhkState, route.Formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindCode binds and validates parameter Code from query.
func (o *GetAuthCallbackParams) bindCode(rawData []string, hasKey bool, formats strfmt.Registry) error {
	if !hasKey {
		return errors.Required("code", "query")
	}
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// AllowEmptyValue: false
	if err := validate.RequiredString("code", "query", raw); err != nil {
		return err
	}

	o.Code = raw

	return nil
}

// bindState binds and validates parameter State from query.
func (o *GetAuthCallbackParams) bindState(rawData []string, hasKey bool, formats strfmt.Registry) error {
	if !hasKey {
		return errors.Required("state", "query")
	}
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// AllowEmptyValue: false
	if err := validate.RequiredString("state", "query", raw); err != nil {
		return err
	}

	o.State = raw

	return nil
}
