package watcher

import "time"

type Config struct {
	Filesystem []*FilesystemConfig
	Minio      []*MinioConfig
}

// See filesystem.go for more.
type FilesystemConfig struct {
	Name         string
	Path         string
	Inotify      bool
	Ignore       string
	TransferType string

	Pipeline         string
	RetentionPeriod  *time.Duration
	StripTopLevelDir bool
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
	Bucket       string
	TransferType string

	Pipeline         string
	RetentionPeriod  *time.Duration
	StripTopLevelDir bool
}
