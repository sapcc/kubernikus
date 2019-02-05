// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// KlusterStatus kluster status
// swagger:model KlusterStatus
type KlusterStatus struct {

	// apiserver
	Apiserver string `json:"apiserver,omitempty"`

	// message
	Message string `json:"message,omitempty"`

	// migrations pending
	MigrationsPending bool `json:"migrationsPending,omitempty"`

	// node pools
	NodePools []NodePoolInfo `json:"nodePools"`

	// phase
	Phase KlusterPhase `json:"phase,omitempty"`

	// spec version
	SpecVersion int64 `json:"specVersion,omitempty"`

	// version
	Version string `json:"version,omitempty"`

	// wormhole
	Wormhole string `json:"wormhole,omitempty"`
}

// Validate validates this kluster status
func (m *KlusterStatus) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateNodePools(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePhase(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *KlusterStatus) validateNodePools(formats strfmt.Registry) error {

	if swag.IsZero(m.NodePools) { // not required
		return nil
	}

	for i := 0; i < len(m.NodePools); i++ {

		if err := m.NodePools[i].Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodePools" + "." + strconv.Itoa(i))
			}
			return err
		}

	}

	return nil
}

func (m *KlusterStatus) validatePhase(formats strfmt.Registry) error {

	if swag.IsZero(m.Phase) { // not required
		return nil
	}

	if err := m.Phase.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("phase")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *KlusterStatus) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *KlusterStatus) UnmarshalBinary(b []byte) error {
	var res KlusterStatus
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
