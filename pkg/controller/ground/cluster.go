package ground

type Cluster struct {
	Certificates *Certificates
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
