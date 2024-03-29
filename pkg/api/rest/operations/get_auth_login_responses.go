// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"
)

// GetAuthLoginFoundCode is the HTTP code returned for type GetAuthLoginFound
const GetAuthLoginFoundCode int = 302

/*
GetAuthLoginFound Redirect

swagger:response getAuthLoginFound
*/
type GetAuthLoginFound struct {
}

// NewGetAuthLoginFound creates GetAuthLoginFound with default headers values
func NewGetAuthLoginFound() *GetAuthLoginFound {

	return &GetAuthLoginFound{}
}

// WriteResponse to the client
func (o *GetAuthLoginFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(302)
}
