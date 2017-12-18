package compute

import (
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/stretchr/testify/assert"
)

func TestCreateOpts(t *testing.T) {
	opts := CreateOpts{
		CreateOpts: servers.CreateOpts{
			Name:           "nase",
			FlavorRef:      "flavor",
			SecurityGroups: []string{"id1", "id2"},
		}}

	data, err := opts.ToServerCreateMap()
	assert.NoError(t, err)
	serverData := data["server"].(map[string]interface{})
	expected := []map[string]interface{}{
		map[string]interface{}{"id": opts.SecurityGroups[0]},
		map[string]interface{}{"id": opts.SecurityGroups[1]},
	}
	assert.Equal(t, expected, serverData["security_groups"])
	assert.Equal(t, opts.Name, serverData["name"])
}
