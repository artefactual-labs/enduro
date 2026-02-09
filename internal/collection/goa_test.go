package collection

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
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

		got := stream.waitEvent(t)
		assert.Equal(t, got.ID, want.ID)
		assert.Equal(t, got.Type, want.Type)

		cancel()
		assert.NilError(t, <-done)
	})
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

func (s *monitorTestStream) waitEvent(t *testing.T) *goacollection.EnduroMonitorUpdate {
	t.Helper()
	select {
	case ev := <-s.events:
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for monitor event")
		return nil
	}
}
