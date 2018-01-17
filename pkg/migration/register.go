package migration

func init() {
	defaultRegistry.migrations = []Migration{
		Init,
	}
}
