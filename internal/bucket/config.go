package bucket

import "time"

type Config struct {
	Filesystem []*FilesystemConfig
	Minio      []*MinioConfig
}

// See filesystem.go for more.
type FilesystemConfig struct {
	Name    string
	Path    string
	RetentionPeriod  *time.Duration
	StripTopLevelDir bool
}

// See minio.go for more.
type MinioConfig struct {
	Name         string
	Region       string
	Endpoint     string
	PathStyle    bool
	Profile      string
	Key          string
	Secret       string
	Token        string
	Bucket       string
	RetentionPeriod  *time.Duration
	StripTopLevelDir bool
}
