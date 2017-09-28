package roles

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/pagination"
)

// ListOpts provides options to filter the List results.
type ListOpts struct {

	// Name filters the response by name.
	Name string `q:"name"`

	// Enabled filters the response by domain_id.
	DomainID string `q:"domain_id"`
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
		return RolePage{pagination.LinkedPageBase{PageResult: r}}
	})
}

func AssignToUserInProject(client *gophercloud.ServiceClient, projectID, userID, roleID string) error {
	url := assignToUserInProjectURL(client, projectID, userID, roleID)

	_, err := client.Put(url, nil, nil, &gophercloud.RequestOpts{OkCodes: []int{204}})
	return err

}
