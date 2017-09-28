package roles

import "github.com/gophercloud/gophercloud"

func listURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("roles")
}
func assignToUserInProjectURL(client *gophercloud.ServiceClient, projectID, userID, roleID string) string {
	return client.ServiceURL("projects", projectID, "users", userID, "roles", roleID)
}
