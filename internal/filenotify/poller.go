package filenotify

import (
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/radovskyb/watcher"
)

// watchWaitTime is the time to wait between file poll loops
const watchWaitTime = 200 * time.Millisecond

// filePoller is used to poll files for changes, especially in cases where fsnotify
// can't be run (e.g. when inotify handles are exhausted)
// filePoller satisfies the FileWatcher interface
type filePoller struct {
	wr *watcher.Watcher

	// events is the channel to listen to for watch events.
	events chan fsnotify.Event

	// errors is the channel to listen to for watch errors.
	errors chan error
}

// loop captures and transform radovskyb's watcher into fsnotify events.
func (w *filePoller) loop() {
	for {
		select {
		case event := <-w.wr.Event:
			var op fsnotify.Op

			switch event.Op {
			case watcher.Create:
				op = fsnotify.Create
			case watcher.Rename:
				fallthrough
			case watcher.Move:
				op = fsnotify.Rename
			default:
				continue
			}

			w.events <- fsnotify.Event{
				Op:   op,
				Name: event.Path,
			}
		case err := <-w.wr.Error:
			w.errors <- err
		case <-w.wr.Closed:
			return
		}
	}
}

func (w *filePoller) Add(name string) error {
	return w.wr.Add(name)
}

func (w *filePoller) Remove(name string) error {
	return w.wr.Remove(name)
}

// Events returns the event channel.
func (w *filePoller) Events() <-chan fsnotify.Event {
	return w.events
}

// Errors returns the errors channel.
func (w *filePoller) Errors() <-chan error {
	return w.errors
}

// Close closes the poller.
func (w *filePoller) Close() error {
	w.wr.Close()

	return nil
}
