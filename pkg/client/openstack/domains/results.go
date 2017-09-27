package domains

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

type domainResult struct {
	gophercloud.Result
}

// GetResult temporarily contains the response from the Get call.
type GetResult struct {
	domainResult
}

// Project is a base unit of ownership.
type Domain struct {
	// Description is the description of the domain.
	Description string `json:"description"`

	// Enabled is whether or not the project is enabled.
	Enabled bool `json:"enabled"`

	// ID is the unique ID of the domain.
	ID string `json:"id"`

	// Name is the name of the domain.
	Name string `json:"name"`
}

// Extract interprets any domainResult as a Domain.
func (r domainResult) Extract() (*Domain, error) {
	var s struct {
		Domain *Domain `json:"domain"`
	}
	err := r.ExtractInto(&s)
	return s.Domain, err
}

// ExtractDomains returns a slice of Domains contained in a single page of results.
func ExtractDomains(r pagination.Page) ([]Domain, error) {
	var s struct {
		Domains []Domain `json:"domains"`
	}
	err := (r.(DomainPage)).ExtractInto(&s)
	return s.Domains, err
}

// DomainPage is a single page of User results.
type DomainPage struct {
	pagination.LinkedPageBase
}

// IsEmpty determines whether or not a DomainPage contains any results.
func (r DomainPage) IsEmpty() (bool, error) {
	domains, err := ExtractDomains(r)
	return len(domains) == 0, err
}

// NextPageURL extracts the "next" link from the links section of the result.
func (r DomainPage) NextPageURL() (string, error) {
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
