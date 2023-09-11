package pipeline

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"go.artefactual.dev/amclient"

	"github.com/artefactual-labs/enduro/internal/pipeline/sync/semaphore"
)

type Config struct {
	ID                 string
	Name               string
	BaseURL            string
	User               string
	Key                string
	TransferDir        string
	TransferLocationID string
	ProcessingDir      string
	ProcessingConfig   string
	StorageServiceURL  string
	Capacity           uint64
	RetryDeadline      *time.Duration
	TransferDeadline   *time.Duration
	Unbag              bool
}

type Pipeline struct {
	logger logr.Logger

	// ID (UUID) of the pipeline. This is not provided by the user but loaded
	// on demand once we have access to the pipeline API.
	ID string

	// A weighted semaphore to limit concurrent use of this pipeline.
	sem *semaphore.Weighted

	// Configuration attributes.
	config *Config

	// The underlying HTTP client used by amclient.
	client *http.Client

	// Pipeline status.
	status          string
	statusUpdatedAt time.Time
	statusLock      sync.RWMutex
}

func NewPipeline(logger logr.Logger, config Config) (*Pipeline, error) {
	config.TransferDir = expandPath(config.TransferDir)
	config.ProcessingDir = expandPath(config.ProcessingDir)

	p := &Pipeline{
		logger: logger,
		sem:    semaphore.NewWeighted(int64(config.Capacity)),
		config: &config,
		client: httpClient(),
	}

	if config.ID != "" {
		p.ID = config.ID
	}

	// init() enriches our record by retrieving the UUID but we still return
	// the the object in case of errors.
	if err := p.init(); err != nil {
		return p, err
	}

	return p, nil
}

// init connects with the pipeline to retrieve its identifier, unless one has
// already been defined via configuration.
func (p *Pipeline) init() error {
	if p.ID != "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	c := p.Client()
	req, err := c.NewRequest(ctx, "GET", "api/v2beta/package/", nil)
	if err != nil {
		return errors.New("error during pipeline identification")
	}

	resp, _ := c.Do(ctx, req, nil)
	if resp == nil || resp.StatusCode != http.StatusNotImplemented {
		return errors.New("error during pipeline identification: unexpected server response")
	}

	id := resp.Header.Get("X-Archivematica-ID")
	if id == "" {
		return errors.New("error during pipeline identification: X-Archivematica-ID header is empty or not present")
	}

	p.ID = id

	return nil
}

// Client returns the Archivematica API client ready for use.
func (p *Pipeline) Client() *amclient.Client {
	return amclient.NewClient(p.client, p.config.BaseURL, p.config.User, p.config.Key)
}

// SSAccess returns the URL and user:key pair needed to access Storage Service.
func (p *Pipeline) SSAccess() (*url.URL, string, error) {
	if p.config.StorageServiceURL == "" {
		return nil, "", errors.New("error parsing storageServiceURL: it is empty")
	}

	u, err := url.Parse(p.config.StorageServiceURL)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing storageServiceURL: %v", err)
	}

	bu := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}

	return bu, u.User.String(), nil
}

// TempFile creates a temporary file in the processing directory.
func (p *Pipeline) TempFile(pattern string) (*os.File, error) {
	if pattern == "" {
		pattern = "blob-*"
	}
	return os.CreateTemp(p.config.ProcessingDir, pattern)
}

func (p *Pipeline) Config() *Config {
	return p.config
}

func (p *Pipeline) TryAcquire() bool {
	return p.sem.TryAcquire(1)
}

func (p *Pipeline) Release() {
	defer func() {
		// sem.Release panics if we release more than is held which is expected
		// to happen when the workflow continues processing after a process
		// kill where state is lost.
		if err := recover(); err != nil {
			p.logger.Info("Pipeline lock release failed", "err", err)
		}
	}()

	p.sem.Release(1)
}

func (p *Pipeline) Capacity() (size, cur int64) {
	return p.sem.Capacity()
}

// loadStatus looks up the status of the pipeline using the HTTP client.
// TODO: find a better way to ping the API.
func (p *Pipeline) loadStatus(ctx context.Context) string {
	_, _, err := p.Client().ProcessingConfig.List(ctx)
	if err != nil {
		return "unavailable"
	}
	return "active"
}

func (p *Pipeline) Status(ctx context.Context) string {
	const ttl = time.Second * 10
	p.statusLock.RLock()
	if time.Since(p.statusUpdatedAt) < ttl {
		status := p.status
		p.statusLock.RUnlock()
		return status
	}
	p.statusLock.RUnlock()
	p.statusLock.Lock()
	defer p.statusLock.Unlock()
	if time.Since(p.statusUpdatedAt) >= ttl {
		p.status = p.loadStatus(ctx)
		p.statusUpdatedAt = time.Now()
	}
	return p.status
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
