package watcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

func TestCompletedDirs(t *testing.T) {
	c := watcher.Config{
		Filesystem: []*watcher.FilesystemConfig{
			{CompletedDir: ""},
			nil,
			{CompletedDir: "/tmp/test-1"},
			{CompletedDir: "/tmp/test-2"},
			{CompletedDir: "./test-3"},
		},
	}

	wd, _ := os.Getwd()
	assert.DeepEqual(t, c.CompletedDirs(), []string{
		"/tmp/test-1",
		"/tmp/test-2",
		filepath.Join(wd, "test-3"),
	})
}

func TestConfigUnmarshalsS3Watcher(t *testing.T) {
	path := filepath.Join(t.TempDir(), "enduro.toml")
	err := os.WriteFile(path, []byte(`
[[watcher.s3]]
name = "dev-s3"
eventSource = "redis"
eventFormat = "minio"
redisAddress = "redis://127.0.0.1:7470"
redisList = "object-events"
endpoint = "http://127.0.0.1:7460"
pathStyle = true
key = "key"
secret = "secret"
region = "us-west-1"
bucket = "sips"
pipeline = "am"
`), 0o600)
	assert.NilError(t, err)

	type config struct {
		Watcher watcher.Config
	}

	v := viper.New()
	v.SetConfigFile(path)
	err = v.ReadInConfig()
	assert.NilError(t, err)

	var c config
	err = v.Unmarshal(&c)
	assert.NilError(t, err)

	assert.DeepEqual(t, c.Watcher.S3, []*watcher.S3Config{
		{
			Name:         "dev-s3",
			Region:       "us-west-1",
			Endpoint:     "http://127.0.0.1:7460",
			PathStyle:    true,
			Key:          "key",
			Secret:       "secret",
			Bucket:       "sips",
			EventSource:  "redis",
			EventFormat:  "minio",
			RedisAddress: "redis://127.0.0.1:7470",
			RedisList:    "object-events",
			Pipeline:     []string{"am"},
		},
	})
}
