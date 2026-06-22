package collection

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	temporalsdk_mocks "go.temporal.io/sdk/mocks"
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

func TestGoaDelete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		rowsAffected int64
		execErr      error
		wantErr      error
		wantNotFound bool
		wantEvent    bool
	}{
		"deletes existing collection": {
			rowsAffected: 1,
			wantEvent:    true,
		},
		"returns not found when no row is deleted": {
			rowsAffected: 0,
			wantNotFound: true,
		},
		"returns database error": {
			rowsAffected: 1,
			execErr:      errTestDB,
			wantErr:      errTestDB,
		},
		"does not cancel workflow": {
			rowsAffected: 1,
			wantEvent:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			recorder := newExecRecorderDB(t)
			recorder.rowsAffected = tc.rowsAffected
			recorder.execErr = tc.execErr

			client := &temporalsdk_mocks.Client{}
			events := NewEventService()
			sub, err := events.Subscribe(ctx)
			assert.NilError(t, err)
			defer sub.Close()

			svc := NewService(testLogger(), recorder.db, client, "", nil)
			svc.events = events

			err = svc.Goa().Delete(ctx, &goacollection.DeletePayload{ID: 42})

			assert.Equal(t, recorder.execQuery, "DELETE FROM collection WHERE id = (?)")
			assert.DeepEqual(t, recorder.execArgs, []any{int64(42)})

			assertGoaServiceErr(t, err, tc.wantErr, tc.wantNotFound)
			assertCollectionEvent(t, sub, tc.wantEvent, EventTypeCollectionDeleted, 42)
			client.AssertExpectations(t)
		})
	}
}

func TestGoaCancel(t *testing.T) {
	t.Parallel()

	errTemporal := errors.New("temporal error")
	createdAt := time.Date(2026, time.June, 11, 10, 0, 0, 0, time.UTC)
	row := &Collection{
		ID:         42,
		Name:       "collection",
		WorkflowID: "processing-workflow-04e9257e-ac59-442c-a037-7504ea5ebf3f",
		RunID:      "74795d4e-4530-4dc1-bb7b-7457ef3c9d75",
		Status:     StatusQueued,
		CreatedAt:  createdAt,
	}

	tests := map[string]struct {
		row          *Collection
		queryErr     error
		cancelErr    error
		wantErr      error
		wantNotFound bool
		wantEvent    bool
		wantCancel   bool
	}{
		"cancels workflow for existing collection": {
			row:        row,
			wantEvent:  true,
			wantCancel: true,
		},
		"returns not found when collection is missing": {
			wantNotFound: true,
		},
		"returns database read error": {
			queryErr: errTestDB,
			wantErr:  errTestDB,
		},
		"returns temporal cancellation error": {
			row:        row,
			cancelErr:  errTemporal,
			wantErr:    errTemporal,
			wantCancel: true,
		},
		"does not delete collection row": {
			row:        row,
			wantEvent:  true,
			wantCancel: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			recorder := newExecRecorderDB(t)
			recorder.row = tc.row
			recorder.queryErr = tc.queryErr

			client := &temporalsdk_mocks.Client{}
			if tc.wantCancel {
				client.On(
					"CancelWorkflow",
					mock.Anything,
					row.WorkflowID,
					row.RunID,
				).Return(tc.cancelErr).Once()
			}

			events := NewEventService()
			sub, err := events.Subscribe(ctx)
			assert.NilError(t, err)
			defer sub.Close()

			svc := NewService(testLogger(), recorder.db, client, "", nil)
			svc.events = events

			err = svc.Goa().Cancel(ctx, &goacollection.CancelPayload{ID: 42})

			assert.Equal(t, recorder.execQuery, "")
			assert.DeepEqual(t, recorder.queryArgs, []any{int64(42)})

			assertGoaServiceErr(t, err, tc.wantErr, tc.wantNotFound)
			assertCollectionEvent(t, sub, tc.wantEvent, EventTypeCollectionUpdated, 42)
			client.AssertExpectations(t)
		})
	}
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

func TestCollectionGoaSummary(t *testing.T) {
	t.Parallel()

	location := time.FixedZone("CEST", 2*60*60)
	storedAt := time.Date(2026, time.June, 17, 12, 30, 0, 0, location)
	col := Collection{
		ID:         42,
		Name:       "collection",
		WorkflowID: "processing-workflow-04e9257e-ac59-442c-a037-7504ea5ebf3f",
		RunID:      "74795d4e-4530-4dc1-bb7b-7457ef3c9d75",
		TransferID: "a5581c4f-c3f7-45c2-b756-787ab9669479",
		AIPID:      "0f83f8f8-79df-4851-a89d-a4e61e9ef112",
		OriginalID: "original-identifier",
		PipelineID: "d964fcd2-7f3f-4640-9068-edcaacf0411b",
		Status:     StatusDone,
		CreatedAt:  storedAt.Add(-time.Hour),
		StartedAt: sql.NullTime{
			Time:  storedAt.Add(-30 * time.Minute),
			Valid: true,
		},
		CompletedAt: sql.NullTime{
			Time:  storedAt,
			Valid: true,
		},
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

	assert.DeepEqual(t, got, &goacollection.EnduroStoredCollection{
		ID:          42,
		Name:        new("collection"),
		WorkflowID:  new("processing-workflow-04e9257e-ac59-442c-a037-7504ea5ebf3f"),
		RunID:       new("74795d4e-4530-4dc1-bb7b-7457ef3c9d75"),
		TransferID:  new("a5581c4f-c3f7-45c2-b756-787ab9669479"),
		AipID:       new("0f83f8f8-79df-4851-a89d-a4e61e9ef112"),
		OriginalID:  new("original-identifier"),
		PipelineID:  new("d964fcd2-7f3f-4640-9068-edcaacf0411b"),
		Status:      "done",
		CreatedAt:   "2026-06-17T09:30:00Z",
		StartedAt:   new("2026-06-17T10:00:00Z"),
		CompletedAt: new("2026-06-17T10:30:00Z"),
	})
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

func assertGoaServiceErr(t *testing.T, err, wantErr error, wantNotFound bool) {
	t.Helper()

	if wantNotFound {
		var notFound *goacollection.CollectionNotfound
		assert.Assert(t, errors.As(err, &notFound))
		assert.Equal(t, notFound.ID, uint(42))
		return
	}

	if wantErr != nil {
		assert.ErrorIs(t, err, wantErr)
		return
	}

	assert.NilError(t, err)
}

func assertCollectionEvent(t *testing.T, sub Subscription, want bool, eventType string, id uint) {
	t.Helper()

	select {
	case got := <-sub.C():
		assert.Assert(t, want, "unexpected collection event")
		assert.Equal(t, got.Type, eventType)
		assert.Equal(t, got.ID, id)
	default:
		assert.Assert(t, !want, "expected collection event")
	}
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
