// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

// GetAuthCallbackReader is a Reader for the GetAuthCallback structure.
type GetAuthCallbackReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetAuthCallbackReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetAuthCallbackOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetAuthCallbackDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetAuthCallbackOK creates a GetAuthCallbackOK with default headers values
func NewGetAuthCallbackOK() *GetAuthCallbackOK {
	return &GetAuthCallbackOK{}
}

/*
GetAuthCallbackOK describes a response with status code 200, with default header values.

OK
*/
type GetAuthCallbackOK struct {
	Payload *models.GetAuthCallbackOKBody
}

// IsSuccess returns true when this get auth callback o k response has a 2xx status code
func (o *GetAuthCallbackOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this get auth callback o k response has a 3xx status code
func (o *GetAuthCallbackOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this get auth callback o k response has a 4xx status code
func (o *GetAuthCallbackOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this get auth callback o k response has a 5xx status code
func (o *GetAuthCallbackOK) IsServerError() bool {
	return false
}

// IsCode returns true when this get auth callback o k response a status code equal to that given
func (o *GetAuthCallbackOK) IsCode(code int) bool {
	return code == 200
}

func (o *GetAuthCallbackOK) Error() string {
	return fmt.Sprintf("[GET /auth/callback][%d] getAuthCallbackOK  %+v", 200, o.Payload)
}

func (o *GetAuthCallbackOK) String() string {
	return fmt.Sprintf("[GET /auth/callback][%d] getAuthCallbackOK  %+v", 200, o.Payload)
}

func (o *GetAuthCallbackOK) GetPayload() *models.GetAuthCallbackOKBody {
	return o.Payload
}

func (o *GetAuthCallbackOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.GetAuthCallbackOKBody)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetAuthCallbackDefault creates a GetAuthCallbackDefault with default headers values
func NewGetAuthCallbackDefault(code int) *GetAuthCallbackDefault {
	return &GetAuthCallbackDefault{
		_statusCode: code,
	}
}

/*
GetAuthCallbackDefault describes a response with status code -1, with default header values.

Error
*/
type GetAuthCallbackDefault struct {
	_statusCode int

	Payload *models.Error
}

// Code gets the status code for the get auth callback default response
func (o *GetAuthCallbackDefault) Code() int {
	return o._statusCode
}

// IsSuccess returns true when this get auth callback default response has a 2xx status code
func (o *GetAuthCallbackDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this get auth callback default response has a 3xx status code
func (o *GetAuthCallbackDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this get auth callback default response has a 4xx status code
func (o *GetAuthCallbackDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this get auth callback default response has a 5xx status code
func (o *GetAuthCallbackDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this get auth callback default response a status code equal to that given
func (o *GetAuthCallbackDefault) IsCode(code int) bool {
	return o._statusCode == code
}

func (o *GetAuthCallbackDefault) Error() string {
	return fmt.Sprintf("[GET /auth/callback][%d] GetAuthCallback default  %+v", o._statusCode, o.Payload)
}

func (o *GetAuthCallbackDefault) String() string {
	return fmt.Sprintf("[GET /auth/callback][%d] GetAuthCallback default  %+v", o._statusCode, o.Payload)
}

func (o *GetAuthCallbackDefault) GetPayload() *models.Error {
	return o.Payload
}

func (o *GetAuthCallbackDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
