// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

// GetClusterInfoOKCode is the HTTP code returned for type GetClusterInfoOK
const GetClusterInfoOKCode int = 200

/*
GetClusterInfoOK OK

swagger:response getClusterInfoOK
*/
type GetClusterInfoOK struct {

	/*
	  In: Body
	*/
	Payload *models.KlusterInfo `json:"body,omitempty"`
}

// NewGetClusterInfoOK creates GetClusterInfoOK with default headers values
func NewGetClusterInfoOK() *GetClusterInfoOK {

	return &GetClusterInfoOK{}
}

// WithPayload adds the payload to the get cluster info o k response
func (o *GetClusterInfoOK) WithPayload(payload *models.KlusterInfo) *GetClusterInfoOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get cluster info o k response
func (o *GetClusterInfoOK) SetPayload(payload *models.KlusterInfo) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetClusterInfoOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*
GetClusterInfoDefault Error

swagger:response getClusterInfoDefault
*/
type GetClusterInfoDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetClusterInfoDefault creates GetClusterInfoDefault with default headers values
func NewGetClusterInfoDefault(code int) *GetClusterInfoDefault {
	if code <= 0 {
		code = 500
	}

	return &GetClusterInfoDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the get cluster info default response
func (o *GetClusterInfoDefault) WithStatusCode(code int) *GetClusterInfoDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the get cluster info default response
func (o *GetClusterInfoDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the get cluster info default response
func (o *GetClusterInfoDefault) WithPayload(payload *models.Error) *GetClusterInfoDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get cluster info default response
func (o *GetClusterInfoDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetClusterInfoDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
