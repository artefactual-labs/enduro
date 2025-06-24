package watcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"gocloud.dev/blob"
)

var (
	ErrWatchTimeout   = errors.New("watcher timed out")
	ErrBucketMismatch = errors.New("bucket mismatch")
)

type Watcher interface {
	// Watch waits until a blob is dispatched.
	Watch(ctx context.Context) (*BlobEvent, error)

	// OpenBucket returns the bucket where the blobs can be found.
	OpenBucket(ctx context.Context) (*blob.Bucket, error)

	// Pipelines retunrs the names of the subset of pipelines to be used.
	Pipelines() []string
	RetentionPeriod() *time.Duration
	CompletedDir() string
	StripTopLevelDir() bool
	RejectDuplicates() bool
	ExcludeHiddenFiles() bool
	TransferType() string

	// Full path of the watched bucket when available, empty string otherwise.
	Path() string

	fmt.Stringer // It should return the name of the watcher.
}

type commonWatcherImpl struct {
	name               string
	pipeline           []string
	retentionPeriod    *time.Duration
	completedDir       string
	stripTopLevelDir   bool
	rejectDuplicates   bool
	excludeHiddenFiles bool
	transferType       string
}

func (w *commonWatcherImpl) String() string {
	return w.name
}

func (w *commonWatcherImpl) Pipelines() []string {
	return w.pipeline
}

func (w *commonWatcherImpl) RetentionPeriod() *time.Duration {
	return w.retentionPeriod
}

func (w *commonWatcherImpl) CompletedDir() string {
	return w.completedDir
}

func (w *commonWatcherImpl) StripTopLevelDir() bool {
	return w.stripTopLevelDir
}

func (w *commonWatcherImpl) RejectDuplicates() bool {
	return w.rejectDuplicates
}

func (w *commonWatcherImpl) ExcludeHiddenFiles() bool {
	return w.excludeHiddenFiles
}

func (w *commonWatcherImpl) TransferType() string {
	return w.transferType
}

type Service interface {
	// Watchers return all known watchers.
	Watchers() []Watcher

	// Return a watcher given its name.
	ByName(name string) (Watcher, error)

	// Download blob given an event.
	Download(ctx context.Context, w io.Writer, watcherName, key string) error

	// Delete blob given an event.
	Delete(ctx context.Context, watcherName, key string) error

	// Dipose blob into the completedDir directory.
	Dispose(ctx context.Context, watcherName, key string) error
}

type serviceImpl struct {
	watchers map[string]Watcher
	mu       sync.RWMutex
}

var _ Service = (*serviceImpl)(nil)

func New(ctx context.Context, c *Config) (*serviceImpl, error) {
	watchers := map[string]Watcher{}

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

	ww := []Watcher{}
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

func (svc *serviceImpl) ByName(name string) (Watcher, error) {
	return svc.watcher(name)
}

func (svc *serviceImpl) Download(ctx context.Context, writer io.Writer, watcherName, key string) error {
	w, err := svc.watcher(watcherName)
	if err != nil {
		return err
	}

	bucket, err := w.OpenBucket(ctx)
	if err != nil {
		return fmt.Errorf("error opening bucket: %w", err)
	}
	defer bucket.Close()

	reader, err := bucket.NewReader(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("error creating reader: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("error copying blob: %w", err)
	}

	return nil
}

func (svc *serviceImpl) Delete(ctx context.Context, watcherName, key string) error {
	w, err := svc.watcher(watcherName)
	if err != nil {
		return err
	}

	bucket, err := w.OpenBucket(ctx)
	if err != nil {
		return fmt.Errorf("error opening bucket: %w", err)
	}
	defer bucket.Close()

	// Exceptionally, a filesystem-based watcher may be dealing with a
	// directory instead of a regular fileblob.
	var fi os.FileInfo
	if bucket.As(&fi) && fi.IsDir() {
		fw, ok := w.(*filesystemWatcher)
		if !ok {
			return fmt.Errorf("error removing directory: %s", err)
		}
		return fw.RemoveAll(key)
	}

	return bucket.Delete(ctx, key)
}

func (svc *serviceImpl) Dispose(ctx context.Context, watcherName, key string) error {
	w, err := svc.watcher(watcherName)
	if err != nil {
		return err
	}

	fw, ok := w.(*filesystemWatcher)
	if !ok {
		return fmt.Errorf("not available in this type of watcher: %s", err)
	}

	return fw.Dispose(key)
}
