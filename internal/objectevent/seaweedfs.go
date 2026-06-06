package objectevent

import (
	"fmt"
	"path"
	"strings"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

const seaweedFSEventTypeCreate = "create"

type seaweedFSEvent struct {
	Key       string                `json:"key"`
	EventType string                `json:"event_type"`
	Message   seaweedFSEventMessage `json:"message"`
}

type seaweedFSEventMessage struct {
	NewEntry *seaweedFSEntry `json:"new_entry"`
}

type seaweedFSEntry struct {
	IsDirectory bool `json:"is_directory"`
}

func enduroEventFromSeaweedFS(event seaweedFSEvent, bucketsPath string) (*watcher.EnduroEvent, bool, error) {
	if event.EventType != seaweedFSEventTypeCreate {
		return nil, false, nil
	}
	if event.Message.NewEntry == nil {
		return nil, false, fmt.Errorf("create event missing new_entry")
	}
	if event.Message.NewEntry.IsDirectory {
		return nil, false, nil
	}

	bucket, key, err := objectFromSeaweedFSPath(event.Key, bucketsPath)
	if err != nil {
		return nil, false, err
	}

	return &watcher.EnduroEvent{
		Version: "1",
		Type:    watcher.EnduroEventTypeObjectCreated,
		Bucket:  bucket,
		Key:     key,
		Source:  "seaweedfs",
	}, true, nil
}

func objectFromSeaweedFSPath(key, bucketsPath string) (string, string, error) {
	key = "/" + strings.TrimLeft(strings.TrimSpace(key), "/")
	bucketsPath = path.Clean("/" + strings.TrimSpace(bucketsPath))

	prefix := bucketsPath + "/"
	if !strings.HasPrefix(key, prefix) {
		return "", "", fmt.Errorf("event key %q is outside buckets path %q", key, bucketsPath)
	}

	parts := strings.SplitN(strings.TrimPrefix(key, prefix), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("event key %q does not include bucket and object key", key)
	}

	return parts[0], parts[1], nil
}
