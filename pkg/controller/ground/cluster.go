package ground

import (
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
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

func NewCluster(kluster *v1.Kluster, authURL string) (*Cluster, error) {
	cluster := &Cluster{
		Certificates: &Certificates{},
		API: API{
			IngressHost: kluster.Spec.KubernikusInfo.Server,
		},
		OpenStack: OpenStack{
			AuthURL:    authURL,
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

	if err := cluster.Certificates.populateForSatellite(kluster.Spec.Name, kluster.Spec.KubernikusInfo.Server); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func (c Cluster) WriteConfig(persister ConfigPersister) error {
	return persister.WriteConfig(c)
}
