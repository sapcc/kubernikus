package ground

type Cluster struct {
	Certificates *Certificates
}

func NewCluster(name string) (*Cluster, error) {
	cluster := &Cluster{}
	if certs, err := newCertificates(name); err != nil {
		return cluster, err
	} else {
		cluster.Certificates = certs
	}

	return cluster, nil
}

func (c Cluster) WriteConfig(persister ConfigPersister) error {
	return persister.WriteConfig(c)
}
