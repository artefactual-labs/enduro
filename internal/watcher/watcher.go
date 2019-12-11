package watcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"gocloud.dev/blob"
)

var ErrWatchTimeout = errors.New("watcher timed out")

type Watcher interface {
	// Watch waits until a blob is dispatched.
	Watch(ctx context.Context) (*BlobEvent, error)

	// OpenBucket returns the bucket where the blobs can be found.
	OpenBucket(ctx context.Context, event *BlobEvent) (*blob.Bucket, error)

	// Every watcher targets a pipeline.
	Pipeline() string

	RetentionPeriod() *time.Duration

	StripTopLevelDir() bool

	fmt.Stringer // It should return the name of the watcher.
}

type commonWatcherImpl struct {
	name             string
	pipeline         string
	retentionPeriod  *time.Duration
	stripTopLevelDir bool
}

func (w *commonWatcherImpl) String() string {
	return w.name
}

func (w *commonWatcherImpl) Pipeline() string {
	return w.pipeline
}

func (w *commonWatcherImpl) RetentionPeriod() *time.Duration {
	return w.retentionPeriod
}

func (w *commonWatcherImpl) StripTopLevelDir() bool {
	return w.stripTopLevelDir
}

//go:generate mockgen  -destination=./fake/mock_watcher.go -package=fake github.com/artefactual-labs/enduro/internal/watcher Service

type Service interface {
	// Watchers return all known watchers.
	Watchers() []Watcher

	// Download blob given an event.
	Download(ctx context.Context, w io.Writer, e *BlobEvent) error

	// Delete blob given an event.
	Delete(ctx context.Context, e *BlobEvent) error
}

type serviceImpl struct {
	watchers map[string]Watcher
	mu       sync.RWMutex
}

var _ Service = (*serviceImpl)(nil)

func New(ctx context.Context, c *Config) (*serviceImpl, error) {
	var watchers = map[string]Watcher{}

	for _, item := range c.Minio {
		w, err := NewMinioWatcher(ctx, item)
		if err != nil {
			return nil, err
		}

		watchers[item.Name] = w
	}

	for _, item := range c.Filesystem {
		w, err := NewFilesystemWatcher(ctx, item)
		if err != nil {
			return nil, err
		}

		watchers[item.Name] = w
	}

	if len(watchers) == 0 {
		return nil, errors.New("there are not watchers configured")
	}

	return &serviceImpl{watchers: watchers}, nil
}

func (svc *serviceImpl) Watchers() []Watcher {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	var ww = []Watcher{}
	for _, item := range svc.watchers {
		ww = append(ww, item)
	}

	return ww
}

func (svc *serviceImpl) watcher(name string) (Watcher, error) {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	w, ok := svc.watchers[name]
	if !ok {
		return nil, fmt.Errorf("error loading watcher: unknown watcher %s", name)
	}

	return w, nil
}

func (svc *serviceImpl) Download(ctx context.Context, writer io.Writer, event *BlobEvent) error {
	w, err := svc.watcher(event.WatcherName)
	if err != nil {
		return err
	}

	bucket, err := w.OpenBucket(ctx, event)
	if err != nil {
		return fmt.Errorf("error opening bucket: %w", err)
	}
	defer bucket.Close()

	reader, err := bucket.NewReader(ctx, event.Key, nil)
	if err != nil {
		return fmt.Errorf("error creating reader: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("error copying blob: %w", err)
	}

	return nil
}

func (svc *serviceImpl) Delete(ctx context.Context, event *BlobEvent) error {
	w, err := svc.watcher(event.WatcherName)
	if err != nil {
		return err
	}

	bucket, err := w.OpenBucket(ctx, event)
	if err != nil {
		return fmt.Errorf("error opening bucket: %w", err)
	}
	defer bucket.Close()

	return bucket.Delete(ctx, event.Key)
}
