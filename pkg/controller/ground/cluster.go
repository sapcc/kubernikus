package ground

import (
	"fmt"

	"github.com/Masterminds/goutils"
	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
)

type Cluster struct {
	Certificates *Certificates `yaml:"certs"`
	API          API           `yaml:"api,omitempty"`
	OpenStack    OpenStack
}

type API struct {
	IngressHost  string `yaml:"ingressHost,omitempty"`
	IngressClass string `yaml:"ingressClass,omitempty"`
}

type OpenStack struct {
	AuthURL    string `yaml:"authURL"`
	Username   string
	Password   string
	DomainName string `yaml:"domainName,omitempty"`
	ProjectID  string `yaml:"projectID,omitempty"`
	Region     string `yaml:"region,omitempty"`
	LBSubnetID string `yaml:"lbSubnetID,omitempty"`
	RouterID   string `yaml:"routerID,omitempty"`
}

func NewCluster(name, domain string) (*Cluster, error) {
	cluster := &Cluster{
		Certificates: &Certificates{},
		API: API{
			IngressHost: fmt.Sprintf("%s.%s", name, domain),
		},
	}

	if err := cluster.Certificates.populateForSatellite(name, domain); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func (c Cluster) WriteConfig(persister ConfigPersister) error {
	return persister.WriteConfig(c)
}

func (c *Cluster) DiscoverValues(name, projectID string, oclient openstack.Client) error {
	if c.OpenStack.Username == "" {
		c.OpenStack.Username = fmt.Sprintf("kubernikus-%s", name)
	}
	var err error
	if c.OpenStack.Password == "" {
		if c.OpenStack.Password, err = goutils.RandomAscii(20); err != nil {
			return fmt.Errorf("Failed to generate password: %s", err)
		}
	}
	if c.OpenStack.DomainName == "" {
		c.OpenStack.DomainName = "Default"
	}
	if c.OpenStack.ProjectID == "" {
		c.OpenStack.ProjectID = projectID
	}
	if c.OpenStack.RouterID == "" || c.OpenStack.LBSubnetID == "" {
		routers, err := oclient.GetRouters(projectID)
		if err != nil {
			return fmt.Errorf("Couldn't get routers for project %s: %s", projectID, err)
		}

		glog.V(2).Infof("Found routers for project %s: %#v", projectID, routers)

		if !(len(routers) == 1 && len(routers[0].Subnets) == 1) {
			return fmt.Errorf("Project needs to contain a router with exactly one subnet")
		}

		if c.OpenStack.RouterID == "" {
			c.OpenStack.RouterID = routers[0].ID
		}
		if c.OpenStack.LBSubnetID == "" {
			c.OpenStack.LBSubnetID = routers[0].Subnets[0].ID
		}
	}
	return nil
}
