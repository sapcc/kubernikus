package etcd

import (
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"

	v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

const (
	BackupStorageContainerBase = "kubernikus-etcd-backup"
)

func DefaultStorageContainer(kluster *v1.Kluster) string {
	return fmt.Sprintf("%s-%s-%s", BackupStorageContainerBase, kluster.Spec.Name, kluster.GetUID())
}

func SetObjectStorageExpiration(providerClient *gophercloud.ProviderClient, containerName string, expiration time.Duration) error {
	storageClient, err := openstack.NewObjectStorageV1(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}

	objectsListOpts := objects.ListOpts{
		Full: false,
	}
	allPages, err := objects.List(storageClient, containerName, objectsListOpts).AllPages()
	if err != nil {
		return err
	}

	allObjects, err := objects.ExtractNames(allPages)
	if err != nil {
		return err
	}

	updateOpts := objects.UpdateOpts{
		DeleteAfter: int(expiration.Seconds()),
	}

	for _, object := range allObjects {
		_, err := objects.Update(storageClient, containerName, object, updateOpts).Extract()
		if err != nil {
			return err
		}
	}

	return nil
}
