package watcher

type Config struct {
	Filesystem []*FilesystemConfig
	Minio      []*MinioConfig
}

// See filesystem.go for more.
type FilesystemConfig struct {
	Name    string
	Path    string
	Inotify bool

	Pipeline string
}

// See minio.go for more.
type MinioConfig struct {
	Name         string
	RedisAddress string
	RedisList    string
	Region       string
	Endpoint     string
	PathStyle    bool
	Profile      string
	Key          string
	Secret       string
	Token        string

	Pipeline string
}
