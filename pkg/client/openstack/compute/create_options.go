package compute

import (
	"errors"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type CreateOpts struct {
	servers.CreateOpts
}

func (opts CreateOpts) ToServerCreateMap() (map[string]interface{}, error) {
	data, err := opts.CreateOpts.ToServerCreateMap()
	if err != nil {
		return nil, err
	}
	if _, ok := data["server"]; !ok {
		return nil, errors.New("Expected field `server` not found")
	}

	serverData, ok := data["server"].(map[string]interface{})
	if !ok {
		return nil, errors.New("Field `server` not of expected type")
	}
	securityGroups := make([]map[string]interface{}, len(opts.SecurityGroups))
	for i, groupID := range opts.SecurityGroups {
		securityGroups[i] = map[string]interface{}{"id": groupID}
	}
	serverData["security_groups"] = securityGroups
	return data, nil
}
