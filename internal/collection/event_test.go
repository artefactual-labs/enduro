package collection_test

import (
	"context"
	"testing"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/artefactual-labs/enduro/internal/collection"
)

func TestEventService(t *testing.T) {
	t.Run("Subscribe", func(t *testing.T) {
		ctx := context.Background()
		s := collection.NewEventService()

		subA, err := s.Subscribe(ctx)
		if err != nil {
			t.Fatal(err)
		}

		subB, err := s.Subscribe(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Publish event to both users
		s.PublishEvent(&goacollection.EnduroMonitorUpdate{ID: 1})

		// Verify both subscribers received the update.
		select {
		case <-subA.C():
		default:
			t.Fatal("expected event")
		}

		select {
		case <-subB.C():
		default:
			t.Fatal("expected event")
		}
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		ctx := context.Background()
		s := collection.NewEventService()

		sub, err := s.Subscribe(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Publish event & close.
		s.PublishEvent(&goacollection.EnduroMonitorUpdate{ID: 1})
		if err := sub.Close(); err != nil {
			t.Fatal(err)
		}

		// Verify event is still received.
		select {
		case <-sub.C():
		default:
			t.Fatal("expected event")
		}

		// Ensure channel is now closed.
		if _, ok := <-sub.C(); ok {
			t.Fatal("expected closed channel")
		}

		// Ensure unsubscribing twice is ok.
		if err := sub.Close(); err != nil {
			t.Fatal(err)
		}
	})
}
