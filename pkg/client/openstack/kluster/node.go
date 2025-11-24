package kluster

import (
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/extendedstatus"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

type Node struct {
	servers.Server
	extendedstatus.ServerExtendedStatusExt
}

func (n *Node) Starting() bool {
	// https://github.com/openstack/nova/blob/be3a66781f7fd58e5c5c0fe89b33f8098cfb0f0d/nova/objects/fields.py#L884
	if n.TaskState == "spawning" || n.TaskState == "scheduling" || n.TaskState == "networking" || n.TaskState == "block_device_mapping" {
		return true
	}

	if n.TaskState != "" {
		return false
	}

	if n.VmState == "building" {
		return true
	}

	return false
}

func (n *Node) Stopping() bool {
	if n.TaskState == "spawning" || n.TaskState == "scheduling" || n.TaskState == "networking" || n.TaskState == "block_device_mapping" || n.TaskState == "image_uploading" || n.TaskState == "rebooting" || n.TaskState == "reboot_started" {
		return false
	}

	if n.TaskState != "" {
		return true
	}

	return false
}

func (n *Node) Running() bool {
	if n.Starting() || n.Stopping() {
		return false
	}

	// 0: NOSTATE
	// 1: RUNNING
	// 3: PAUSED
	// 4: SHUTDOWN
	// 6: CRASHED
	// 7: SUSPENDED
	if n.PowerState > 1 {
		return false
	}

	//ACTIVE = 'active'
	//BUILDING = 'building'
	//PAUSED = 'paused'
	//SUSPENDED = 'suspended'
	//STOPPED = 'stopped'
	//RESCUED = 'rescued'
	//RESIZED = 'resized'
	//SOFT_DELETED = 'soft-delete'
	//DELETED = 'deleted'
	//ERROR = 'error'
	//SHELVED = 'shelved'
	//SHELVED_OFFLOADED = 'shelved_offloaded'
	if n.VmState == "active" {
		return true
	}

	return false
}

func (n *Node) Erroring() bool {
	if n.TaskState == "deleting" {
		return false
	}

	return n.VmState == "error"
}

func (n *Node) GetID() string {
	return n.ID
}

func (n *Node) GetName() string {
	return n.Name
}

func (n *Node) GetCreated() time.Time {
	return n.Created
}

func (n *Node) GetSecurityGroupNames() []string {
	names := []string{}
	for _, s := range n.SecurityGroups {
		if name, ok := s["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}
