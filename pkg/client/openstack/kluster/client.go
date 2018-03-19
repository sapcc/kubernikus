package kluster

import (
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
	"k8s.io/client-go/tools/cache"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack/compute"
)

type KlusterClient interface {
	CreateNode(*models.NodePool, []byte) (string, error)
	DeleteNode(string) error
	ListNodes(*models.NodePool) ([]Node, error)
}

type klusterClient struct {
	NetworkClient  *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient

	Kluster   *v1.Kluster
	NodeStore cache.Store
}

type cachedNodesEntry struct {
	Kluster *v1.Kluster
	Pool    *models.NodePool
	Nodes   []Node
}

func CachedNodesKeyFunc(obj interface{}) (string, error) {
	entry, ok := obj.(cachedNodesEntry)
	if !ok {
		return "", fmt.Errorf("unexpected object in cache")
	}
	return fmt.Sprintf("%v-%v", entry.Kluster.Spec.Name, entry.Pool.Name), nil
}

func NewKlusterClient(network, compute, identity *gophercloud.ServiceClient, kluster *v1.Kluster) KlusterClient {
	var client KlusterClient
	client = &klusterClient{
		NetworkClient:  network,
		ComputeClient:  compute,
		IdentityClient: identity,
		Kluster:        kluster,
		NodeStore:      cache.NewTTLStore(CachedNodesKeyFunc, 1*time.Minute),
	}

	return client
}

func (c *klusterClient) CreateNode(pool *models.NodePool, userData []byte) (string, error) {
	var name string
	name = SimpleNameGenerator.GenerateName(fmt.Sprintf("%v-%v-", c.Kluster.Spec.Name, pool.Name))

	configDrive := true
	server, err := compute.Create(c.ComputeClient, servers.CreateOpts{
		Name:           name,
		FlavorName:     pool.Flavor,
		ImageName:      pool.Image,
		Networks:       []servers.Network{servers.Network{UUID: c.Kluster.Spec.Openstack.NetworkID}},
		UserData:       userData,
		ServiceClient:  c.ComputeClient,
		SecurityGroups: []string{c.Kluster.Spec.Openstack.SecurityGroupName},
		ConfigDrive:    &configDrive,
	}).Extract()

	if err != nil {
		return "", err
	}

	return server.ID, nil
}

func (c *klusterClient) DeleteNode(id string) (err error) {
	err = servers.Delete(c.ComputeClient, id).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

func (c *klusterClient) ListNodes(pool *models.NodePool) (nodes []Node, err error) {
	obj, exists, err := c.NodeStore.Get(cachedNodesEntry{c.Kluster, pool, nil})
	if err != nil {
		return nil, err
	}
	if exists {
		return obj.(cachedNodesEntry).Nodes, nil
	}

	prefix := fmt.Sprintf("%v-%v-", c.Kluster.Spec.Name, pool.Name)
	err = servers.List(c.ComputeClient, servers.ListOpts{Name: prefix}).EachPage(func(page pagination.Page) (bool, error) {
		nodes, err = ExtractServers(page)
		if err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	err = c.NodeStore.Add(cachedNodesEntry{c.Kluster, pool, nodes})
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func ExtractServers(r pagination.Page) ([]Node, error) {
	var s []Node
	err := servers.ExtractServersInto(r, &s)
	return s, err
}
