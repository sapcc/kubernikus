// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

// GetClusterKubeadmSecretOKCode is the HTTP code returned for type GetClusterKubeadmSecretOK
const GetClusterKubeadmSecretOKCode int = 200

/*
GetClusterKubeadmSecretOK OK

swagger:response getClusterKubeadmSecretOK
*/
type GetClusterKubeadmSecretOK struct {

	/*
	  In: Body
	*/
	Payload *models.KubeadmSecret `json:"body,omitempty"`
}

// NewGetClusterKubeadmSecretOK creates GetClusterKubeadmSecretOK with default headers values
func NewGetClusterKubeadmSecretOK() *GetClusterKubeadmSecretOK {

	return &GetClusterKubeadmSecretOK{}
}

// WithPayload adds the payload to the get cluster kubeadm secret o k response
func (o *GetClusterKubeadmSecretOK) WithPayload(payload *models.KubeadmSecret) *GetClusterKubeadmSecretOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get cluster kubeadm secret o k response
func (o *GetClusterKubeadmSecretOK) SetPayload(payload *models.KubeadmSecret) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetClusterKubeadmSecretOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*
GetClusterKubeadmSecretDefault Error

swagger:response getClusterKubeadmSecretDefault
*/
type GetClusterKubeadmSecretDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetClusterKubeadmSecretDefault creates GetClusterKubeadmSecretDefault with default headers values
func NewGetClusterKubeadmSecretDefault(code int) *GetClusterKubeadmSecretDefault {
	if code <= 0 {
		code = 500
	}

	return &GetClusterKubeadmSecretDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the get cluster kubeadm secret default response
func (o *GetClusterKubeadmSecretDefault) WithStatusCode(code int) *GetClusterKubeadmSecretDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the get cluster kubeadm secret default response
func (o *GetClusterKubeadmSecretDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the get cluster kubeadm secret default response
func (o *GetClusterKubeadmSecretDefault) WithPayload(payload *models.Error) *GetClusterKubeadmSecretDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get cluster kubeadm secret default response
func (o *GetClusterKubeadmSecretDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetClusterKubeadmSecretDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
