// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// KlusterSpec kluster spec
//
// swagger:model KlusterSpec
type KlusterSpec struct {

	// advertise address
	AdvertiseAddress string `json:"advertiseAddress,omitempty"`

	// advertise port
	AdvertisePort int64 `json:"advertisePort"`

	// audit
	// Enum: [elasticsearch swift http stdout]
	Audit *string `json:"audit,omitempty"`

	// backup
	// Enum: [on off externalAWS]
	Backup string `json:"backup,omitempty"`

	// CIDR Range for Pods in the cluster. Can not be updated.
	// Pattern: ^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2])))?$
	ClusterCIDR *string `json:"clusterCIDR,omitempty"`

	// custom c n i
	CustomCNI bool `json:"customCNI,omitempty"`

	// dashboard
	Dashboard *bool `json:"dashboard"`

	// dex
	Dex *bool `json:"dex"`

	// dns address
	// Pattern: ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$
	DNSAddress string `json:"dnsAddress,omitempty"`

	// dns domain
	DNSDomain string `json:"dnsDomain,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// no cloud
	NoCloud bool `json:"noCloud,omitempty"`

	// node pools
	NodePools []NodePool `json:"nodePools"`

	// oidc
	Oidc *OIDC `json:"oidc,omitempty"`

	// openstack
	Openstack OpenstackSpec `json:"openstack,omitempty"`

	// seed kubeadm
	SeedKubeadm bool `json:"seedKubeadm,omitempty"`

	// CIDR Range for Services in the cluster. Can not be updated.
	// Pattern: ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2]))$
	ServiceCIDR string `json:"serviceCIDR,omitempty"`

	// SSH public key that is injected into spawned nodes.
	// Max Length: 10000
	SSHPublicKey string `json:"sshPublicKey,omitempty"`

	// Kubernetes version
	// Pattern: ^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$
	Version string `json:"version,omitempty"`
}

// Validate validates this kluster spec
func (m *KlusterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAudit(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateBackup(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateClusterCIDR(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDNSAddress(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNodePools(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOidc(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOpenstack(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateServiceCIDR(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSSHPublicKey(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVersion(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

var klusterSpecTypeAuditPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["elasticsearch","swift","http","stdout"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		klusterSpecTypeAuditPropEnum = append(klusterSpecTypeAuditPropEnum, v)
	}
}

const (

	// KlusterSpecAuditElasticsearch captures enum value "elasticsearch"
	KlusterSpecAuditElasticsearch string = "elasticsearch"

	// KlusterSpecAuditSwift captures enum value "swift"
	KlusterSpecAuditSwift string = "swift"

	// KlusterSpecAuditHTTP captures enum value "http"
	KlusterSpecAuditHTTP string = "http"

	// KlusterSpecAuditStdout captures enum value "stdout"
	KlusterSpecAuditStdout string = "stdout"
)

// prop value enum
func (m *KlusterSpec) validateAuditEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, klusterSpecTypeAuditPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *KlusterSpec) validateAudit(formats strfmt.Registry) error {
	if swag.IsZero(m.Audit) { // not required
		return nil
	}

	// value enum
	if err := m.validateAuditEnum("audit", "body", *m.Audit); err != nil {
		return err
	}

	return nil
}

var klusterSpecTypeBackupPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["on","off","externalAWS"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		klusterSpecTypeBackupPropEnum = append(klusterSpecTypeBackupPropEnum, v)
	}
}

const (

	// KlusterSpecBackupOn captures enum value "on"
	KlusterSpecBackupOn string = "on"

	// KlusterSpecBackupOff captures enum value "off"
	KlusterSpecBackupOff string = "off"

	// KlusterSpecBackupExternalAWS captures enum value "externalAWS"
	KlusterSpecBackupExternalAWS string = "externalAWS"
)

// prop value enum
func (m *KlusterSpec) validateBackupEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, klusterSpecTypeBackupPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *KlusterSpec) validateBackup(formats strfmt.Registry) error {
	if swag.IsZero(m.Backup) { // not required
		return nil
	}

	// value enum
	if err := m.validateBackupEnum("backup", "body", m.Backup); err != nil {
		return err
	}

	return nil
}

func (m *KlusterSpec) validateClusterCIDR(formats strfmt.Registry) error {
	if swag.IsZero(m.ClusterCIDR) { // not required
		return nil
	}

	if err := validate.Pattern("clusterCIDR", "body", *m.ClusterCIDR, `^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2])))?$`); err != nil {
		return err
	}

	return nil
}

func (m *KlusterSpec) validateDNSAddress(formats strfmt.Registry) error {
	if swag.IsZero(m.DNSAddress) { // not required
		return nil
	}

	if err := validate.Pattern("dnsAddress", "body", m.DNSAddress, `^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`); err != nil {
		return err
	}

	return nil
}

func (m *KlusterSpec) validateNodePools(formats strfmt.Registry) error {
	if swag.IsZero(m.NodePools) { // not required
		return nil
	}

	for i := 0; i < len(m.NodePools); i++ {

		if err := m.NodePools[i].Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodePools" + "." + strconv.Itoa(i))
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nodePools" + "." + strconv.Itoa(i))
			}
			return err
		}

	}

	return nil
}

func (m *KlusterSpec) validateOidc(formats strfmt.Registry) error {
	if swag.IsZero(m.Oidc) { // not required
		return nil
	}

	if m.Oidc != nil {
		if err := m.Oidc.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("oidc")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("oidc")
			}
			return err
		}
	}

	return nil
}

func (m *KlusterSpec) validateOpenstack(formats strfmt.Registry) error {
	if swag.IsZero(m.Openstack) { // not required
		return nil
	}

	if err := m.Openstack.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("openstack")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("openstack")
		}
		return err
	}

	return nil
}

func (m *KlusterSpec) validateServiceCIDR(formats strfmt.Registry) error {
	if swag.IsZero(m.ServiceCIDR) { // not required
		return nil
	}

	if err := validate.Pattern("serviceCIDR", "body", m.ServiceCIDR, `^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\/([0-9]|[1-2][0-9]|3[0-2]))$`); err != nil {
		return err
	}

	return nil
}

func (m *KlusterSpec) validateSSHPublicKey(formats strfmt.Registry) error {
	if swag.IsZero(m.SSHPublicKey) { // not required
		return nil
	}

	if err := validate.MaxLength("sshPublicKey", "body", m.SSHPublicKey, 10000); err != nil {
		return err
	}

	return nil
}

func (m *KlusterSpec) validateVersion(formats strfmt.Registry) error {
	if swag.IsZero(m.Version) { // not required
		return nil
	}

	if err := validate.Pattern("version", "body", m.Version, `^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`); err != nil {
		return err
	}

	return nil
}

// ContextValidate validate this kluster spec based on the context it is used
func (m *KlusterSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateNodePools(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOidc(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOpenstack(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *KlusterSpec) contextValidateNodePools(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.NodePools); i++ {

		if err := m.NodePools[i].ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodePools" + "." + strconv.Itoa(i))
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nodePools" + "." + strconv.Itoa(i))
			}
			return err
		}

	}

	return nil
}

func (m *KlusterSpec) contextValidateOidc(ctx context.Context, formats strfmt.Registry) error {

	if m.Oidc != nil {
		if err := m.Oidc.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("oidc")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("oidc")
			}
			return err
		}
	}

	return nil
}

func (m *KlusterSpec) contextValidateOpenstack(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Openstack.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("openstack")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("openstack")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *KlusterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *KlusterSpec) UnmarshalBinary(b []byte) error {
	var res KlusterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
