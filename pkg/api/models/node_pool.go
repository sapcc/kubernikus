// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// NodePool node pool
// swagger:model NodePool
type NodePool struct {

	// availability zone
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// config
	Config NodePoolConfig `json:"config,omitempty"`

	// flavor
	// Required: true
	Flavor string `json:"flavor"`

	// image
	Image string `json:"image,omitempty"`

	// name
	// Required: true
	// Max Length: 20
	// Pattern: ^[a-z]([a-z0-9]*)?$
	Name string `json:"name"`

	// size
	// Maximum: 127
	// Minimum: 0
	Size int64 `json:"size,omitempty"`
}

// Validate validates this node pool
func (m *NodePool) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateConfig(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateFlavor(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSize(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *NodePool) validateConfig(formats strfmt.Registry) error {

	if swag.IsZero(m.Config) { // not required
		return nil
	}

	if err := m.Config.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("config")
		}
		return err
	}

	return nil
}

func (m *NodePool) validateFlavor(formats strfmt.Registry) error {

	if err := validate.RequiredString("flavor", "body", string(m.Flavor)); err != nil {
		return err
	}

	return nil
}

func (m *NodePool) validateName(formats strfmt.Registry) error {

	if err := validate.RequiredString("name", "body", string(m.Name)); err != nil {
		return err
	}

	if err := validate.MaxLength("name", "body", string(m.Name), 20); err != nil {
		return err
	}

	if err := validate.Pattern("name", "body", string(m.Name), `^[a-z]([a-z0-9]*)?$`); err != nil {
		return err
	}

	return nil
}

func (m *NodePool) validateSize(formats strfmt.Registry) error {

	if swag.IsZero(m.Size) { // not required
		return nil
	}

	if err := validate.MinimumInt("size", "body", int64(m.Size), 0, false); err != nil {
		return err
	}

	if err := validate.MaximumInt("size", "body", int64(m.Size), 127, false); err != nil {
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *NodePool) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *NodePool) UnmarshalBinary(b []byte) error {
	var res NodePool
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
