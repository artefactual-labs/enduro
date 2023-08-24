package metadata

type Config struct {
	ProcessNameMetadata bool
}

func (c Config) IsEnabled() bool {
	return c.ProcessNameMetadata
}
