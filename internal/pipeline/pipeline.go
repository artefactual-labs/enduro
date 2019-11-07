package pipeline

import (
	"errors"
	"fmt"
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

// TODO: PipelineRegistry should return Pipeline objects!

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

// PipelineRegistry is a collection of known pipelines.
type PipelineRegistry struct {
	pipelines map[string]Config
}

func NewPipelineRegistry(configs []Config) *PipelineRegistry {
	pipelines := map[string]Config{}
	for _, cfg := range configs {
		cfg.TransferDir = expandPath(cfg.TransferDir)
		pipelines[cfg.Name] = cfg

	}
	return &PipelineRegistry{
		pipelines: pipelines,
	}
}

func (p PipelineRegistry) Config(name string) (*Config, error) {
	cfg, ok := p.pipelines[name]
	if !ok {
		return nil, errors.New("client not found")
	}

	return &cfg, nil
}

func (p PipelineRegistry) Client(name string) (*amclient.Client, error) {
	cfg, err := p.Config(name)
	if err != nil {
		return nil, fmt.Errorf("error fetching pipeline configuration: %w", err)
	}

	client, err := amclient.New(httpClient(), cfg.BaseURL, cfg.User, cfg.Key)
	if err != nil {
		return nil, fmt.Errorf("error creating Archivematica API client: %w", err)
	}

	return client, nil
}

// TempFile returns a new temporary file inside the processing directory of the
// given pipeline.
func (p PipelineRegistry) TempFile(name string) (*os.File, error) {
	cfg, err := p.Config(name)
	if err != nil {
		return nil, fmt.Errorf("error fetching pipeline configuration: %w", err)
	}

	return ioutil.TempFile(cfg.ProcessingDir, "blob-*")
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
