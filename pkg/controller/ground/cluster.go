package ground

type Cluster struct {
	Certificates *Certificates `yaml:"certs"`
	API          API           `yaml:"api,omitempty"`
	OpenStack    OpenStack
}

type API struct {
	IngressHost string `yaml:"ingressHost,omitempty"`
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

func NewCluster(name string) (*Cluster, error) {
	cluster := &Cluster{
		Certificates: &Certificates{},
	}

	if err := cluster.Certificates.populateForSatellite(name); err != nil {
		return cluster, err
	}

	return cluster, nil
}

func (c Cluster) WriteConfig(persister ConfigPersister) error {
	return persister.WriteConfig(c)
}
