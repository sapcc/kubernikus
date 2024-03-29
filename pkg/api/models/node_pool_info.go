// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// NodePoolInfo node pool info
//
// swagger:model NodePoolInfo
type NodePoolInfo struct {

	// healthy
	Healthy int64 `json:"healthy"`

	// name
	Name string `json:"name,omitempty"`

	// running
	Running int64 `json:"running"`

	// schedulable
	Schedulable int64 `json:"schedulable"`

	// size
	Size int64 `json:"size"`
}

// Validate validates this node pool info
func (m *NodePoolInfo) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this node pool info based on context it is used
func (m *NodePoolInfo) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *NodePoolInfo) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *NodePoolInfo) UnmarshalBinary(b []byte) error {
	var res NodePoolInfo
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
