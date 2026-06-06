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

const (
	S3EventSourceRedis  = "redis"
	S3EventFormatMinio  = "minio"
	S3EventFormatEnduro = "enduro"
)

// s3Watcher implements a Watcher for S3-compatible object storage.
type s3Watcher struct {
	redisClient redis.UniversalClient
	s3Client    *s3.Client
	listName    string
	bucketName  string
	eventFormat string
	*commonWatcherImpl
}

type MinioEventSet struct {
	Event     []MinioEvent
	EventTime string
}

var _ Watcher = (*s3Watcher)(nil)

const redisPopTimeout = time.Second * 2

func NewMinioWatcher(ctx context.Context, config *MinioConfig) (*s3Watcher, error) {
	return NewS3Watcher(ctx, s3ConfigFromMinio(config))
}

func s3ConfigFromMinio(config *MinioConfig) *S3Config {
	if config == nil {
		return nil
	}

	return &S3Config{
		Name:               config.Name,
		Region:             config.Region,
		Endpoint:           config.Endpoint,
		PathStyle:          config.PathStyle,
		Profile:            config.Profile,
		Key:                config.Key,
		Secret:             config.Secret,
		Token:              config.Token,
		Bucket:             config.Bucket,
		EventSource:        S3EventSourceRedis,
		EventFormat:        S3EventFormatMinio,
		RedisAddress:       config.RedisAddress,
		RedisList:          config.RedisList,
		Pipeline:           config.Pipeline,
		RetentionPeriod:    config.RetentionPeriod,
		StripTopLevelDir:   config.StripTopLevelDir,
		RejectDuplicates:   config.RejectDuplicates,
		ExcludeHiddenFiles: config.ExcludeHiddenFiles,
		TransferType:       config.TransferType,
	}
}

func NewS3Watcher(ctx context.Context, config *S3Config) (*s3Watcher, error) {
	if config == nil {
		return nil, errors.New("missing S3 watcher config")
	}
	if config.EventSource != S3EventSourceRedis {
		return nil, fmt.Errorf("unsupported S3 watcher event source %q", config.EventSource)
	}
	if config.EventFormat != S3EventFormatMinio && config.EventFormat != S3EventFormatEnduro {
		return nil, fmt.Errorf("unsupported S3 watcher event format %q", config.EventFormat)
	}

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

	return &s3Watcher{
		redisClient: client,
		listName:    config.RedisList,
		s3Client:    s3Client,
		bucketName:  config.Bucket,
		eventFormat: config.EventFormat,
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

func (w *s3Watcher) Watch(ctx context.Context) (*BlobEvent, error) {
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

func (w *s3Watcher) Path() string {
	return ""
}

func (w *s3Watcher) blpop(ctx context.Context) (*BlobEvent, error) {
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

func (w *s3Watcher) event(blob string) (*BlobEvent, error) {
	switch w.eventFormat {
	case S3EventFormatMinio:
		return w.minioEvent(blob)
	case S3EventFormatEnduro:
		return w.enduroEvent(blob)
	default:
		return nil, fmt.Errorf("unsupported S3 watcher event format %q", w.eventFormat)
	}
}

// minioEvent processes MinIO-specific events delivered via Redis. We expect a
// single item array containing a map of {Event: ..., EvenTime: ...}
func (w *s3Watcher) minioEvent(blob string) (*BlobEvent, error) {
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

func (w *s3Watcher) enduroEvent(blob string) (*BlobEvent, error) {
	var event EnduroEvent
	if err := json.Unmarshal([]byte(blob), &event); err != nil {
		return nil, err
	}
	if event.Version != "1" {
		return nil, fmt.Errorf("error processing item received from Redis list: unsupported Enduro event version %q", event.Version)
	}
	if event.Type != EnduroEventTypeObjectCreated {
		return nil, fmt.Errorf("error processing item received from Redis list: unsupported Enduro event type %q", event.Type)
	}
	if event.Bucket == "" {
		return nil, fmt.Errorf("error processing item received from Redis list: empty bucket")
	}
	if event.Key == "" {
		return nil, fmt.Errorf("error processing item received from Redis list: empty key")
	}

	key, err := url.QueryUnescape(event.Key)
	if err != nil {
		return nil, fmt.Errorf("error processing item received from Redis list: %w", err)
	}

	return NewBlobEventWithBucket(w, event.Bucket, key), nil
}

func (w *s3Watcher) OpenBucket(ctx context.Context) (*blob.Bucket, error) {
	return s3blob.OpenBucketV2(ctx, w.s3Client, w.bucketName, nil)
}
