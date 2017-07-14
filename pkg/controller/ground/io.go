package ground

type ConfigPersister interface {
	WriteConfig(Cluster) error
}
