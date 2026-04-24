package collection

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
)

func TestGoaMonitor(t *testing.T) {
	t.Run("Forwards published events", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		events := NewEventService()
		w := &goaWrapper{
			collectionImpl: &collectionImpl{
				events: events,
			},
		}

		stream := newMonitorTestStream()
		done := make(chan error, 1)
		go func() {
			done <- w.Monitor(ctx, stream)
		}()

		waitForSubscriptions(t, events, 1)

		want := &goacollection.EnduroMonitorUpdate{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			ID:        123,
			Type:      EventTypeCollectionUpdated,
		}
		w.events.PublishEvent(want)

		got := stream.waitEventByType(t, want.Type)
		assert.Equal(t, got.ID, want.ID)
		assert.Equal(t, got.Type, want.Type)

		cancel()
		assert.NilError(t, <-done)
	})
}

func TestRetryModeForCollection(t *testing.T) {
	t.Parallel()

	registry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "recovery-enabled",
			ID:   "pipeline-enabled",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		},
		{
			Name: "recovery-disabled",
			ID:   "pipeline-disabled",
		},
	}, nil, nil)
	assert.NilError(t, err)

	tests := map[string]struct {
		pipelineName string
		col          *Collection
		want         RetryMode
	}{
		"No AIP means full reprocess": {
			pipelineName: "recovery-enabled",
			col:          &Collection{},
			want:         RetryModeFullReprocess,
		},
		"Recovery enabled via pipeline ID": {
			col: &Collection{
				AIPID:      "aip-1",
				PipelineID: "pipeline-enabled",
			},
			want: RetryModeReconcileExistingAIP,
		},
		"Recovery enabled via pipeline name fallback": {
			pipelineName: "recovery-enabled",
			col: &Collection{
				AIPID: "aip-1",
			},
			want: RetryModeReconcileExistingAIP,
		},
		"Recovery disabled keeps full reprocess": {
			col: &Collection{
				AIPID:      "aip-1",
				PipelineID: "pipeline-disabled",
			},
			want: RetryModeFullReprocess,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := retryModeForCollection(registry, tc.pipelineName, tc.col)
			assert.Equal(t, got, tc.want)
		})
	}
}

func TestCollectionGoaDetailIncludesReconciliationFields(t *testing.T) {
	t.Parallel()

	checkedAt := time.Date(2026, time.March, 18, 10, 0, 0, 0, time.UTC)
	aipStoredAt := checkedAt.Add(-time.Hour)

	col := Collection{
		ID:        42,
		AIPID:     "0f83f8f8-79df-4851-a89d-a4e61e9ef112",
		Status:    StatusDone,
		CreatedAt: checkedAt.Add(-2 * time.Hour),
		AIPStoredAt: sql.NullTime{
			Time:  aipStoredAt,
			Valid: true,
		},
		ReconciliationStatus: sql.NullString{
			String: "complete",
			Valid:  true,
		},
		ReconciliationCheckedAt: sql.NullTime{
			Time:  checkedAt,
			Valid: true,
		},
		ReconciliationError: sql.NullString{
			String: "replica lag",
			Valid:  true,
		},
	}

	got := col.GoaDetail()

	assert.Equal(t, got.ID, uint(42))
	assert.Assert(t, got.AipStoredAt != nil)
	assert.Equal(t, *got.AipStoredAt, aipStoredAt.Format(time.RFC3339))
	assert.Assert(t, got.ReconciliationStatus != nil)
	assert.Equal(t, *got.ReconciliationStatus, "complete")
	assert.Assert(t, got.ReconciliationCheckedAt != nil)
	assert.Equal(t, *got.ReconciliationCheckedAt, checkedAt.Format(time.RFC3339))
	assert.Assert(t, got.ReconciliationError != nil)
	assert.Equal(t, *got.ReconciliationError, "replica lag")
}

func TestCollectionGoaSummaryOmitsReconciliationFields(t *testing.T) {
	t.Parallel()

	col := Collection{
		ID:        42,
		Status:    StatusDone,
		CreatedAt: time.Now().UTC(),
		AIPStoredAt: sql.NullTime{
			Time:  time.Now().UTC(),
			Valid: true,
		},
		ReconciliationStatus: sql.NullString{
			String: "partial",
			Valid:  true,
		},
		ReconciliationCheckedAt: sql.NullTime{
			Time:  time.Now().UTC(),
			Valid: true,
		},
		ReconciliationError: sql.NullString{
			String: "replica missing",
			Valid:  true,
		},
	}

	got := col.GoaSummary()

	assert.Equal(t, got.ID, uint(42))
}

func TestNewRetryResult(t *testing.T) {
	t.Parallel()

	got := newRetryResult(RetryModeReconcileExistingAIP)

	assert.Equal(t, got.Mode, string(RetryModeReconcileExistingAIP))
}

func waitForSubscriptions(t *testing.T, events *EventServiceImpl, count int) {
	t.Helper()

	poll.WaitOn(t, func(log poll.LogT) poll.Result {
		events.mu.Lock()
		n := len(events.subs)
		events.mu.Unlock()
		if n == count {
			return poll.Success()
		}
		return poll.Continue("got %d subscriptions, want %d", n, count)
	}, poll.WithTimeout(time.Second), poll.WithDelay(time.Millisecond*5))
}

type monitorTestStream struct {
	events chan *goacollection.EnduroMonitorUpdate
}

func newMonitorTestStream() *monitorTestStream {
	return &monitorTestStream{
		events: make(chan *goacollection.EnduroMonitorUpdate, 8),
	}
}

func (s *monitorTestStream) Send(v *goacollection.EnduroMonitorUpdate) error {
	s.events <- v
	return nil
}

func (s *monitorTestStream) SendWithContext(_ context.Context, v *goacollection.EnduroMonitorUpdate) error {
	return s.Send(v)
}

func (*monitorTestStream) Close() error {
	return nil
}

func (s *monitorTestStream) waitEventByType(t *testing.T, eventType string) *goacollection.EnduroMonitorUpdate {
	t.Helper()

	var matched *goacollection.EnduroMonitorUpdate

	poll.WaitOn(t, func(log poll.LogT) poll.Result {
		for {
			select {
			case ev := <-s.events:
				if ev.Type == eventType {
					matched = ev
					return poll.Success()
				}
			default:
				return poll.Continue("waiting for monitor event type %q", eventType)
			}
		}
	}, poll.WithTimeout(time.Second), poll.WithDelay(time.Millisecond*5))

	return matched
}
