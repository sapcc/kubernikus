package roles

import "github.com/gophercloud/gophercloud/pagination"

// Role represents a Role in an assignment.
type Role struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	DomainID string `json:"domain_id,omitempty"`
}

// ExtractRoles returns a slice of Domains contained in a single page of results.
func ExtractRoles(r pagination.Page) ([]Role, error) {
	var s struct {
		Roles []Role `json:"Roles"`
	}
	err := (r.(RolePage)).ExtractInto(&s)
	return s.Roles, err
}

// RolePage is a single page of User results.
type RolePage struct {
	pagination.LinkedPageBase
}

// IsEmpty determines whether or not a RolePage contains any results.
func (r RolePage) IsEmpty() (bool, error) {
	domains, err := ExtractRoles(r)
	return len(domains) == 0, err
}

// NextPageURL extracts the "next" link from the links section of the result.
func (r RolePage) NextPageURL() (string, error) {
	var s struct {
		Links struct {
			Next     string `json:"next"`
			Previous string `json:"previous"`
		} `json:"links"`
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return s.Links.Next, err
}
