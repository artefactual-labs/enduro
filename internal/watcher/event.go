package watcher

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// BlobEvent is a serializable event that describes a blob.
//
// BlobEvent can be sent over the wire, i.e. they're serializable. Receivers,
// typicially Cadence activities, can download the blob via the service
// implementation in this package.
//
// TODO: use signed URLs to simplify access to buckets?
type BlobEvent struct {
	// Name of the watcher that received this blob.
	WatcherName string

	// Name of the pipeline that the watcher targets.
	PipelineName string

	// Retention period for this blob.
	RetentionPeriod time.Duration

	// Key of the blob.
	Key string

	// Bucket where the blob lives.
	Bucket string `json:"Bucket,omitempty"`
}

func NewBlobEvent(w Watcher, key string) *BlobEvent {
	return &BlobEvent{
		WatcherName:     w.String(),
		PipelineName:    w.Pipeline(),
		RetentionPeriod: w.RetentionPeriod(),
		Key:             key,
	}
}

func NewBlobEventWithBucket(w Watcher, bucket, key string) *BlobEvent {
	e := NewBlobEvent(w, key)
	e.Bucket = bucket
	return e
}

func (e BlobEvent) String() string {
	var key = e.Key

	if e.Bucket != "" {
		key = fmt.Sprintf("%s:%s", e.Bucket, key)
	}

	return fmt.Sprintf("%q:%q", e.WatcherName, key)
}

var uuid4regex = regexp.MustCompile(`[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}`)

func (e BlobEvent) NameUUID() string {
	s := uuid4regex.FindString(e.Key)

	_, err := uuid.Parse(s)
	if err != nil {
		return ""
	}

	return s
}
