package domains

import "github.com/gophercloud/gophercloud"

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

// Extract interprets any projectResults as a Project.
func (r domainResult) Extract() (*Domain, error) {
	var s struct {
		Domain *Domain `json:"domain"`
	}
	err := r.ExtractInto(&s)
	return s.Domain, err
}
