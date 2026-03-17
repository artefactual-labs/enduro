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
	ssclient "go.artefactual.dev/ssclient"

	"github.com/artefactual-labs/enduro/internal/pipeline/sync/semaphore"
)

var ErrRecoveryConfigInvalid = errors.New("invalid recovery configuration")

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
	Recovery           RecoveryConfig
}

type RecoveryConfig struct {
	ReconcileExistingAIP bool
	RequiredLocations    []string
}

func (c Config) Validate() error {
	if !c.Recovery.ReconcileExistingAIP {
		return nil
	}

	if c.StorageServiceURL == "" {
		return fmt.Errorf("%w: storageServiceURL is required when reconcileExistingAIP is enabled", ErrRecoveryConfigInvalid)
	}

	return nil
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
	archivematicaHTTPClient *http.Client

	// The underlying HTTP client used by Storage Service requests.
	storageServiceHTTPClient *http.Client

	// The Storage Service SDK client bound to this pipeline configuration.
	storageServiceClient *ssclient.Client

	// Pipeline status.
	status          string
	statusUpdatedAt time.Time
	statusLock      sync.RWMutex
	TaskQueue       string
}

func NewPipeline(logger logr.Logger, config Config, archivematicaHTTPClient, storageServiceHTTPClient *http.Client) (*Pipeline, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	config.TransferDir = expandPath(config.TransferDir)
	config.ProcessingDir = expandPath(config.ProcessingDir)
	archivematicaHTTPClient = defaultArchivematicaHTTPClient(archivematicaHTTPClient)
	storageServiceHTTPClient = defaultStorageServiceHTTPClient(storageServiceHTTPClient)

	storageServiceClient, err := newStorageServiceClient(config, storageServiceHTTPClient)
	if err != nil {
		return nil, err
	}

	p := &Pipeline{
		logger:                   logger,
		sem:                      semaphore.NewWeighted(int64(config.Capacity)),
		config:                   &config,
		archivematicaHTTPClient:  archivematicaHTTPClient,
		storageServiceHTTPClient: storageServiceHTTPClient,
		storageServiceClient:     storageServiceClient,
		TaskQueue:                TaskQueueName(config.Name),
	}

	if config.ID != "" {
		p.ID = config.ID
	}

	// init() enriches our record by retrieving the UUID, but we still return
	// the object in case of errors.
	if err := p.init(); err != nil {
		return p, err
	}

	return p, nil
}

func TaskQueueName(name string) string {
	return fmt.Sprintf("pipeline-%s", name)
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
	return amclient.NewClient(p.archivematicaHTTPClient, p.config.BaseURL, p.config.User, p.config.Key)
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

func defaultArchivematicaHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return newHTTPClient(10 * time.Second)
	}
	return client
}

func defaultStorageServiceHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return newHTTPClient(30 * time.Second)
	}
	return client
}

func newHTTPClient(timeout time.Duration) *http.Client {
	const (
		dialTimeout      = 5 * time.Second
		handshakeTimeout = 5 * time.Second
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

func newStorageServiceClient(config Config, httpClient *http.Client) (*ssclient.Client, error) {
	if strings.TrimSpace(config.StorageServiceURL) == "" {
		return nil, nil
	}

	baseURL, username, key, err := parseStorageServiceURL(config.StorageServiceURL)
	if err != nil {
		return nil, err
	}

	client, err := ssclient.New(ssclient.Config{
		BaseURL:    strings.TrimRight(baseURL.String(), "/"),
		Username:   username,
		Key:        key,
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("create storage service client: %w", err)
	}

	return client, nil
}

func parseStorageServiceURL(raw string) (*url.URL, string, string, error) {
	if raw == "" {
		return nil, "", "", errors.New("error parsing storageServiceURL: it is empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", "", fmt.Errorf("error parsing storageServiceURL: %v", err)
	}

	baseURL := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}

	password, _ := u.User.Password()
	return cloneURL(baseURL), u.User.Username(), password, nil
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return &url.URL{}
	}

	clone := *u
	return &clone
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
