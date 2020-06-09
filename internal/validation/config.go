package validation

type Config struct {
	ChecksumsCheckEnabled bool
}

func (c Config) IsEnabled() bool {
	return c.ChecksumsCheckEnabled
}
