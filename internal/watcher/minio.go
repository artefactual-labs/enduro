package watcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/redis/go-redis/v9"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

// minioWatcher implements a Watcher for watching lists in Redis.
type minioWatcher struct {
	client   redis.UniversalClient
	sess     *session.Session
	listName string
	bucket   string
	*commonWatcherImpl
}

type MinioEventSet struct {
	Event     []MinioEvent
	EventTime string
}

var _ Watcher = (*minioWatcher)(nil)

const redisPopTimeout = time.Second * 2

func NewMinioWatcher(ctx context.Context, config *MinioConfig) (*minioWatcher, error) {
	opts, err := redis.ParseURL(config.RedisAddress)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

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
		bucket:   config.Bucket,
		commonWatcherImpl: &commonWatcherImpl{
			name:             config.Name,
			pipeline:         config.Pipeline,
			retentionPeriod:  config.RetentionPeriod,
			stripTopLevelDir: config.StripTopLevelDir,
			rejectDuplicates: config.RejectDuplicates,
		},
	}, nil
}

func (w *minioWatcher) Watch(ctx context.Context) (*BlobEvent, error) {
	event, err := w.blpop(ctx)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = ErrWatchTimeout
		}
		return nil, err
	}
	if event.Bucket != w.bucket {
		return nil, ErrBucketMismatch
	}
	return event, nil
}

func (w *minioWatcher) Path() string {
	return ""
}

func (w *minioWatcher) blpop(ctx context.Context) (*BlobEvent, error) {
	val, err := w.client.BLPop(ctx, redisPopTimeout, w.listName).Result()
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
// single item array containing a map of {Event: ..., EvenTime: ...}
func (w *minioWatcher) event(blob string) (*BlobEvent, error) {
	container := []json.RawMessage{}
	if err := json.Unmarshal([]byte(blob), &container); err != nil {
		return nil, err
	}

	var set MinioEventSet
	if err := json.Unmarshal(container[0], &set); err != nil {
		return nil, fmt.Errorf("error procesing item received from Redis list: %w", err)
	}
	if len(set.Event) == 0 {
		return nil, fmt.Errorf("error processing item received from Redis list: empty event list")
	}

	key, err := url.QueryUnescape(set.Event[0].S3.Object.Key)
	if err != nil {
		return nil, fmt.Errorf("error processing item received from Redis list: %w", err)
	}

	return NewBlobEventWithBucket(w, set.Event[0].S3.Bucket.Name, key), nil
}

func (w *minioWatcher) OpenBucket(ctx context.Context) (*blob.Bucket, error) {
	return s3blob.OpenBucket(ctx, w.sess, w.bucket, nil)
}
