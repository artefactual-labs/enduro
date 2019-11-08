package pipeline

import (
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-labs/enduro/internal/amclient"
)

type Config struct {
	Name               string
	BaseURL            string
	User               string
	Key                string
	TransferDir        string
	TransferLocationID string
	ProcessingDir      string
	ProcessingConfig   string
}

type Pipeline struct {
	config *Config
	client *http.Client
}

func NewPipeline(config *Config) *Pipeline {
	config.TransferDir = expandPath(config.TransferDir)
	config.ProcessingDir = expandPath(config.ProcessingDir)

	return &Pipeline{
		config: config,
		client: httpClient(),
	}
}

// Client returns the Archivematica API client ready for use.
func (p Pipeline) Client() *amclient.Client {
	return amclient.NewClient(p.client, p.config.BaseURL, p.config.User, p.config.Key)
}

// TempFile creates a temporary file in the processing directory.
func (p Pipeline) TempFile(pattern string) (*os.File, error) {
	if pattern == "" {
		pattern = "blob-*"
	}
	return ioutil.TempFile(p.config.ProcessingDir, pattern)
}

func (p Pipeline) Config() *Config {
	return p.config
}

func httpClient() *http.Client {
	const (
		dialTimeout      = 5 * time.Second
		handshakeTimeout = 5 * time.Second
		timeout          = 10 * time.Second
	)
	dialer := &net.Dialer{
		Timeout: dialTimeout,
	}
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: handshakeTimeout,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func expandPath(path string) string {
	usr, _ := user.Current()
	dir := usr.HomeDir

	if path == "~" {
		path = dir
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(dir, path[2:])
	}

	return path
}
