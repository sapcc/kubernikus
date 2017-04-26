package ground

type Cluster struct {
	Certificates ClusterCerts
}

func NewCluster(name string) *Cluster {
	cluster := &Cluster{}
	cluster.Certificates = newClusterCerts(name)

	return cluster
}

func (c Cluster) WriteConfig(persister ConfigPersister) error {
	return persister.WriteConfig(c)
}
