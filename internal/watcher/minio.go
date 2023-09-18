package watcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

// minioWatcher implements a Watcher for watching lists in Redis.
type minioWatcher struct {
	redisClient redis.UniversalClient
	s3Client    *s3.Client
	listName    string
	bucketName  string
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

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(config.Profile),
		awsconfig.WithRegion(config.Region),
		func(lo *awsconfig.LoadOptions) error {
			if config.Key != "" && config.Secret != "" {
				lo.Credentials = credentials.StaticCredentialsProvider{
					Value: aws.Credentials{
						AccessKeyID:     config.Key,
						SecretAccessKey: config.Secret,
						SessionToken:    config.Token,
					},
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(awsConfig, func(opts *s3.Options) {
		opts.UsePathStyle = config.PathStyle
		opts.Region = config.Region
		if config.Endpoint != "" {
			opts.BaseEndpoint = &config.Endpoint
		}
	})

	return &minioWatcher{
		redisClient: client,
		listName:    config.RedisList,
		s3Client:    s3Client,
		bucketName:  config.Bucket,
		commonWatcherImpl: &commonWatcherImpl{
			name:               config.Name,
			pipeline:           config.Pipeline,
			retentionPeriod:    config.RetentionPeriod,
			stripTopLevelDir:   config.StripTopLevelDir,
			rejectDuplicates:   config.RejectDuplicates,
			excludeHiddenFiles: config.ExcludeHiddenFiles,
			transferType:       config.TransferType,
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
	if event.Bucket != w.bucketName {
		return nil, ErrBucketMismatch
	}
	return event, nil
}

func (w *minioWatcher) Path() string {
	return ""
}

func (w *minioWatcher) blpop(ctx context.Context) (*BlobEvent, error) {
	val, err := w.redisClient.BLPop(ctx, redisPopTimeout, w.listName).Result()
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
	return s3blob.OpenBucketV2(ctx, w.s3Client, w.bucketName, nil)
}
