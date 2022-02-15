package watcher

import (
	"path/filepath"
	"time"
)

type Config struct {
	Filesystem []*FilesystemConfig
	Minio      []*MinioConfig
}

func (c Config) CompletedDirs() []string {
	dirs := []string{}
	for _, item := range c.Filesystem {
		if item == nil {
			continue
		}
		if item.CompletedDir == "" {
			continue
		}
		if abs, err := filepath.Abs(item.CompletedDir); err == nil {
			dirs = append(dirs, abs)
		}
	}
	return dirs
}

// See filesystem.go for more.
type FilesystemConfig struct {
	Name    string
	Path    string
	Inotify bool
	Ignore  string

	Pipeline         []string
	RetentionPeriod  *time.Duration
	CompletedDir     string
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

	Pipeline         []string
	RetentionPeriod  *time.Duration
	StripTopLevelDir bool
}
