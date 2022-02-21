package watcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/fsnotify/fsnotify"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"

	"github.com/artefactual-labs/enduro/internal/filenotify"
	"github.com/artefactual-labs/enduro/internal/fsutil"
)

// filesystemWatcher implements a Watcher for watching paths in a local filesystem.
type filesystemWatcher struct {
	ctx   context.Context
	fsw   filenotify.FileWatcher
	ch    chan *fsnotify.Event
	path  string
	regex *regexp.Regexp
	*commonWatcherImpl
}

var _ Watcher = (*filesystemWatcher)(nil)

func NewFilesystemWatcher(ctx context.Context, config *FilesystemConfig) (*filesystemWatcher, error) {
	stat, err := os.Stat(config.Path)
	if err != nil {
		return nil, fmt.Errorf("error looking up stat info: %w", err)
	}
	if !stat.IsDir() {
		return nil, errors.New("given path is not a directory")
	}
	abspath, err := filepath.Abs(config.Path)
	if err != nil {
		return nil, fmt.Errorf("error generating absolute path of %s: %v", config.Path, err)
	}

	var regex *regexp.Regexp
	if config.Ignore != "" {
		if regex, err = regexp.Compile(config.Ignore); err != nil {
			return nil, fmt.Errorf("error compiling regular expression (ignore): %v", err)
		}
	}

	if config.CompletedDir != "" && config.RetentionPeriod != nil {
		return nil, errors.New("cannot use completedDir and retentionPeriod simultaneously")
	}

	// The inotify API isn't always available, fall back to polling.
	var fsw filenotify.FileWatcher
	if config.Inotify && runtime.GOOS != "windows" {
		fsw, err = filenotify.New()
	} else {
		fsw, err = filenotify.NewPollingWatcher()
	}
	if err != nil {
		return nil, fmt.Errorf("error creating filesystem watcher: %w", err)
	}

	w := &filesystemWatcher{
		ctx:   ctx,
		fsw:   fsw,
		ch:    make(chan *fsnotify.Event, 100),
		path:  abspath,
		regex: regex,
		commonWatcherImpl: &commonWatcherImpl{
			name:             config.Name,
			pipeline:         config.Pipeline,
			retentionPeriod:  config.RetentionPeriod,
			completedDir:     config.CompletedDir,
			stripTopLevelDir: config.StripTopLevelDir,
		},
	}

	go w.loop()

	if err := fsw.Add(abspath); err != nil {
		return nil, fmt.Errorf("error configuring filesystem watcher: %w", err)
	}

	return w, nil
}

func (w *filesystemWatcher) loop() {
	for {
		select {
		case event, ok := <-w.fsw.Events():
			if !ok {
				continue
			}
			if event.Op != fsnotify.Create && event.Op != fsnotify.Rename {
				continue
			}
			if path, err := filepath.Abs(event.Name); err != nil || path == w.path {
				continue
			}
			if w.regex != nil && w.regex.MatchString(filepath.Base(event.Name)) {
				continue
			}
			w.ch <- &event
		case _, ok := <-w.fsw.Errors():
			if !ok {
				continue
			}
		case <-w.ctx.Done():
			_ = w.fsw.Close()
			close(w.ch)
			return
		}
	}
}

func (w *filesystemWatcher) Watch(ctx context.Context) (*BlobEvent, error) {
	fsevent, ok := <-w.ch
	if !ok {
		return nil, ErrWatchTimeout
	}
	info, err := os.Stat(fsevent.Name)
	if err != nil {
		return nil, fmt.Errorf("error in file stat check: %s", err)
	}
	rel, err := filepath.Rel(w.path, fsevent.Name)
	if err != nil {
		return nil, fmt.Errorf("error generating relative path of fsvent.Name %s - %w", fsevent.Name, err)
	}
	return NewBlobEvent(w, rel, info.IsDir()), nil
}

func (w *filesystemWatcher) Path() string {
	return w.path
}

func (w *filesystemWatcher) OpenBucket(context.Context) (*blob.Bucket, error) {
	return fileblob.OpenBucket(w.path, nil)
}

func (w *filesystemWatcher) RemoveAll(key string) error {
	return os.RemoveAll(filepath.Join(w.path, key))
}

func (w *filesystemWatcher) Dispose(key string) error {
	if w.completedDir == "" {
		return nil
	}

	src := filepath.Join(w.path, key)
	dst := filepath.Join(w.completedDir, key)

	return fsutil.Move(src, dst)
}
