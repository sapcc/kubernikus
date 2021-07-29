// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/swag"
)

// Principal principal
// swagger:model Principal
type Principal struct {

	// account id
	Account string `json:"account,omitempty"`

	// account name
	AccountName string `json:"account_name,omitempty"`

	// user's domain name
	Domain string `json:"domain,omitempty"`

	// userid
	ID string `json:"id,omitempty"`

	// username
	Name string `json:"name,omitempty"`

	// list of roles the user has in the given scope
	Roles []string `json:"roles"`
}

// Validate validates this principal
func (m *Principal) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Principal) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Principal) UnmarshalBinary(b []byte) error {
	var res Principal
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
