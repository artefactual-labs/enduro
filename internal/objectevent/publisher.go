package objectevent

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

type Publisher interface {
	Publish(ctx context.Context, event *watcher.EnduroEvent) error
}

type RedisPublisher struct {
	client redis.UniversalClient
	list   string
}

func NewRedisPublisher(redisAddress, list string) (*RedisPublisher, error) {
	opts, err := redis.ParseURL(redisAddress)
	if err != nil {
		return nil, err
	}

	return &RedisPublisher{
		client: redis.NewClient(opts),
		list:   list,
	}, nil
}

func (p *RedisPublisher) Publish(ctx context.Context, event *watcher.EnduroEvent) error {
	if event == nil {
		return errors.New("missing object event")
	}

	msg, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.client.RPush(ctx, p.list, msg).Err()
}

func (p *RedisPublisher) Close() error {
	return p.client.Close()
}
