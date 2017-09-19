package ground

import (
	"fmt"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/config"
)

type Cluster struct {
	Certificates *Certificates `yaml:"certs"`
	API          API           `yaml:"api,omitempty"`
	OpenStack    OpenStack
	Kubernikus   Kubernikus
}

type API struct {
	IngressHost  string `yaml:"ingressHost,omitempty"`
	IngressClass string `yaml:"ingressClass,omitempty"`
	WormholeHost string `yaml:"wormholeHost,omitempty"`
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

type Kubernikus struct {
	BootstrapToken string `yaml:"bootstrapToken,omitempty"`
}

func NewCluster(kluster *v1.Kluster, config config.Config) (*Cluster, error) {
	cluster := &Cluster{
		Certificates: &Certificates{},
		API: API{
			IngressHost:  fmt.Sprintf("%v.%v", kluster.GetName(), config.Kubernikus.Domain),
			WormholeHost: fmt.Sprintf("%v-wormhole.%v", kluster.GetName(), config.Kubernikus.Domain),
		},
		OpenStack: OpenStack{
			AuthURL:    kluster.Spec.OpenstackInfo.AuthURL,
			Username:   kluster.Spec.OpenstackInfo.Username,
			Password:   kluster.Spec.OpenstackInfo.Password,
			DomainName: kluster.Spec.OpenstackInfo.Domain,
			ProjectID:  kluster.Spec.OpenstackInfo.ProjectID,
			LBSubnetID: kluster.Spec.OpenstackInfo.LBSubnetID,
			RouterID:   kluster.Spec.OpenstackInfo.RouterID,
		},
		Kubernikus: Kubernikus{
			BootstrapToken: kluster.Spec.KubernikusInfo.BootstrapToken,
		},
	}

	if err := cluster.Certificates.populateForSatellite(kluster.GetName(), config); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func (c Cluster) WriteConfig(persister ConfigPersister) error {
	return persister.WriteConfig(c)
}
