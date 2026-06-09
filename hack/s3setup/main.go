package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

func main() {
	var (
		endpoint        = flag.String("endpoint", "", "S3 endpoint URL")
		region          = flag.String("region", "us-west-1", "S3 region")
		bucket          = flag.String("bucket", "", "S3 bucket")
		accessKey       = flag.String("access-key", "", "S3 access key")
		secretKey       = flag.String("secret-key", "", "S3 secret key")
		notificationARN = flag.String("notification-arn", "", "MinIO notification target ARN")
	)
	flag.Parse()

	if *endpoint == "" || *bucket == "" || *accessKey == "" || *secretKey == "" {
		fmt.Fprintln(os.Stderr, "endpoint, bucket, access-key, and secret-key are required")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(*region),
		awsconfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     *accessKey,
				SecretAccessKey: *secretKey,
			},
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load AWS config: %v\n", err)
		os.Exit(1)
	}

	client := s3.NewFromConfig(cfg, func(opts *s3.Options) {
		opts.BaseEndpoint = endpoint
		opts.UsePathStyle = true
		opts.Region = *region
	})

	if err := retry(ctx, func() error {
		return setupBucket(ctx, client, *bucket, *notificationARN)
	}); err != nil {
		fmt.Fprintf(os.Stderr, "setup bucket: %v\n", err)
		os.Exit(1)
	}
}

func setupBucket(ctx context.Context, client *s3.Client, bucket, notificationARN string) error {
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &bucket})
	if err != nil && !bucketExists(err) {
		return fmt.Errorf("create bucket: %w", err)
	}

	if notificationARN == "" {
		return nil
	}

	_, err = client.PutBucketNotificationConfiguration(ctx, &s3.PutBucketNotificationConfigurationInput{
		Bucket: &bucket,
		NotificationConfiguration: &types.NotificationConfiguration{
			QueueConfigurations: []types.QueueConfiguration{
				{
					QueueArn: &notificationARN,
					Events:   []types.Event{types.EventS3ObjectCreated},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("put bucket notification: %w", err)
	}

	return nil
}

func bucketExists(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	switch apiErr.ErrorCode() {
	case "BucketAlreadyOwnedByYou", "BucketAlreadyExists":
		return true
	default:
		return false
	}
}

func retry(ctx context.Context, fn func() error) error {
	var lastErr error
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		if err := fn(); err != nil {
			lastErr = err
		} else {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %w", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}
