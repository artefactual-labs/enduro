package objectevent

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

type fakePublisher struct {
	err    error
	events []*watcher.EnduroEvent
}

func (p *fakePublisher) Publish(ctx context.Context, event *watcher.EnduroEvent) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, event)
	return nil
}

func TestSeaweedFSEventsPublishesObjectCreatedEvent(t *testing.T) {
	pub := &fakePublisher{}
	srv := HTTPServer(logr.Discard(), &Config{
		Listen:      "127.0.0.1:0",
		BucketsPath: "/buckets",
	}, pub)

	req := httptest.NewRequest(http.MethodPost, "/seaweedfs/events", bytes.NewBufferString(`{
		"key": "/buckets/sips/path/to/transfer.zip",
		"event_type": "create",
		"message": {
			"new_entry": {
				"name": "transfer.zip",
				"is_directory": false
			}
		}
	}`))
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusAccepted)
	assert.DeepEqual(t, pub.events, []*watcher.EnduroEvent{
		{
			Version: "1",
			Type:    watcher.EnduroEventTypeObjectCreated,
			Bucket:  "sips",
			Key:     "path/to/transfer.zip",
			Source:  "seaweedfs",
		},
	})
}

func TestSeaweedFSEventsIgnoresUnsupportedEvents(t *testing.T) {
	pub := &fakePublisher{}
	srv := HTTPServer(logr.Discard(), &Config{
		Listen:      "127.0.0.1:0",
		BucketsPath: "/buckets",
	}, pub)

	req := httptest.NewRequest(http.MethodPost, "/seaweedfs/events", bytes.NewBufferString(`{
		"key": "/buckets/sips/path/to/transfer.zip",
		"event_type": "delete",
		"message": {}
	}`))
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusNoContent)
	assert.Assert(t, len(pub.events) == 0)
}

func TestSeaweedFSEventsReturnsBadRequestOnInvalidPayload(t *testing.T) {
	pub := &fakePublisher{}
	srv := HTTPServer(logr.Discard(), &Config{
		Listen:      "127.0.0.1:0",
		BucketsPath: "/buckets",
	}, pub)

	req := httptest.NewRequest(http.MethodPost, "/seaweedfs/events", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusBadRequest)
	assert.Assert(t, len(pub.events) == 0)
}

func TestSeaweedFSEventsReturnsServerErrorOnPublishError(t *testing.T) {
	pub := &fakePublisher{err: errors.New("boom")}
	srv := HTTPServer(logr.Discard(), &Config{
		Listen:      "127.0.0.1:0",
		BucketsPath: "/buckets",
	}, pub)

	req := httptest.NewRequest(http.MethodPost, "/seaweedfs/events", bytes.NewBufferString(`{
		"key": "/buckets/sips/path/to/transfer.zip",
		"event_type": "create",
		"message": {
			"new_entry": {
				"is_directory": false
			}
		}
	}`))
	rec := httptest.NewRecorder()

	srv.Handler.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusInternalServerError)
}
