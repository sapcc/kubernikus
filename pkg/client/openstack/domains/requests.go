package domains

import "github.com/gophercloud/gophercloud"

// Get retrieves details on a single project, by ID.
func Get(client *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = client.Get(getURL(client, id), &r.Body, nil)
	return
}
