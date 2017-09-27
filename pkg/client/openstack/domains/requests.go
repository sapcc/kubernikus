package domains

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// Get retrieves details on a single project, by ID.
func Get(client *gophercloud.ServiceClient, id string) (r GetResult) {
	_, r.Err = client.Get(getURL(client, id), &r.Body, nil)
	return
}

// ListOpts provides options to filter the List results.
type ListOpts struct {

	// Name filters the response by name.
	Name string `q:"name"`

	// Enabled filters the response by enabled domains.
	Enabled *bool `q:"enabled"`
}

func List(client *gophercloud.ServiceClient, opts *ListOpts) pagination.Pager {
	url := listURL(client)
	if opts != nil {
		query, err := gophercloud.BuildQueryString(opts)
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query.String()
	}
	return pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return DomainPage{pagination.LinkedPageBase{PageResult: r}}
	})
}
