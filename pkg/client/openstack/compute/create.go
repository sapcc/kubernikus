package compute

import (
	"encoding/json"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type createError struct {
}

func (ce *createError) Error400(e gophercloud.ErrUnexpectedResponseCode) error {

	var response struct {
		BadRequest struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"badRequest"`
	}
	if err := json.Unmarshal(e.Body, &response); err != nil || response.BadRequest.Message == "" {
		return fmt.Errorf("Failed to parse response from compute: %s", string(e.Body))
	}
	return fmt.Errorf("Response from compute: %d %s", e.Actual, response.BadRequest.Message)
}

func (ce createError) Error() string {
	//Unused, but we need to satisfy the error interface
	return "Failed to create server. This shouldn't be returned ever."
}

func (ce *createError) Error403(e gophercloud.ErrUnexpectedResponseCode) error {
	var response struct {
		Forbidden struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"forbidden"`
	}
	if err := json.Unmarshal(e.Body, &response); err != nil || response.Forbidden.Message == "" {
		return fmt.Errorf("Failed to parse response from compute: %s", string(e.Body))
	}
	return fmt.Errorf("Response from compute: %d %s", e.Actual, response.Forbidden.Message)
}

// Create requests a server to be provisioned to the user in the current tenant.
func Create(client *gophercloud.ServiceClient, opts servers.CreateOptsBuilder) (r servers.CreateResult) {
	reqBody, err := opts.ToServerCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	_, r.Err = client.Post(client.ServiceURL("servers"), reqBody, &r.Body, &gophercloud.RequestOpts{
		ErrorContext: &createError{},
	})
	return
}
