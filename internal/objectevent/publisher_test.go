package objectevent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

func TestRedisPublisherPublishesEnduroEvent(t *testing.T) {
	redis := miniredis.RunT(t)

	pub, err := NewRedisPublisher(fmt.Sprintf("redis://%s", redis.Addr()), "object-events")
	assert.NilError(t, err)
	defer pub.Close()

	err = pub.Publish(context.Background(), &watcher.EnduroEvent{
		Version: "1",
		Type:    watcher.EnduroEventTypeObjectCreated,
		Bucket:  "sips",
		Key:     "transfer.zip",
		Source:  "seaweedfs",
	})
	assert.NilError(t, err)

	items, err := redis.List("object-events")
	assert.NilError(t, err)
	assert.Assert(t, len(items) == 1)

	var event watcher.EnduroEvent
	err = json.Unmarshal([]byte(items[0]), &event)
	assert.NilError(t, err)
	assert.DeepEqual(t, event, watcher.EnduroEvent{
		Version: "1",
		Type:    watcher.EnduroEventTypeObjectCreated,
		Bucket:  "sips",
		Key:     "transfer.zip",
		Source:  "seaweedfs",
	})
}
