package watcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-redis/redis/v7"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

// minioWatcher implements a Watcher for watching lists in Redis.
type minioWatcher struct {
	client   *redis.Client
	sess     *session.Session
	listName string
	*commonWatcherImpl
}

var _ Watcher = (*minioWatcher)(nil)

const redisPopTimeout = time.Second * 2

func NewMinioWatcher(ctx context.Context, config *MinioConfig) (*minioWatcher, error) {
	opts, err := redis.ParseURL(config.RedisAddress)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts).WithContext(ctx)

	sessOpts := session.Options{}
	sessOpts.Config.WithRegion(config.Region)
	if config.Profile != "" {
		sessOpts.Profile = config.Profile
	}
	if config.Endpoint != "" {
		sessOpts.Config.WithEndpoint(config.Endpoint)
	}
	if config.PathStyle {
		sessOpts.Config.WithS3ForcePathStyle(config.PathStyle)
	}
	if config.Key != "" && config.Secret != "" {
		sessOpts.Config.WithCredentials(credentials.NewStaticCredentials(config.Key, config.Secret, config.Token))
	}
	sess, err := session.NewSessionWithOptions(sessOpts)
	if err != nil {
		return nil, err
	}

	return &minioWatcher{
		client:   client,
		sess:     sess,
		listName: config.RedisList,
		commonWatcherImpl: &commonWatcherImpl{
			name:            config.Name,
			pipeline:        config.Pipeline,
			retentionPeriod: config.RetentionPeriod,
		},
	}, nil
}

func (w *minioWatcher) Watch(ctx context.Context) (*BlobEvent, error) {
	event, err := w.blpop(ctx)
	if errors.Is(err, redis.Nil) {
		return nil, ErrWatchTimeout
	}
	return event, err
}

func (w *minioWatcher) blpop(ctx context.Context) (*BlobEvent, error) {
	val, err := w.client.BLPop(redisPopTimeout, w.listName).Result()
	if err != nil {
		return nil, fmt.Errorf("error retrieving from Redis list: %w", err)
	}

	event, err := w.event(val[1])
	if err != nil {
		return nil, fmt.Errorf("ereror processing item received: %w", err)
	}

	return event, nil
}

// event processes Minio-specific events delivered via Redis. We expect a
// two-item array: 0 = time of the event, 1 = single-item array containing the
// actual event.
func (w *minioWatcher) event(blob string) (*BlobEvent, error) {
	container := []json.RawMessage{}
	if err := json.Unmarshal([]byte(blob), &container); err != nil {
		return nil, err
	}

	var (
		_      = container[0]   // Event time.
		events = container[1]   // Actual events.
		evs    = []MinioEvent{} // We only expect one really.
	)
	if err := json.Unmarshal(events, &evs); err != nil {
		return nil, fmt.Errorf("error procesing item received from Redis list: %w", err)
	}
	if len(evs) == 0 {
		return nil, fmt.Errorf("error processing item received from Redis list: empty event list")
	}

	return NewBlobEventWithBucket(w, evs[0].S3.Bucket.Name, evs[0].S3.Object.Key), nil
}

func (w *minioWatcher) OpenBucket(ctx context.Context, event *BlobEvent) (*blob.Bucket, error) {
	return s3blob.OpenBucket(ctx, w.sess, event.Bucket, nil)
}
