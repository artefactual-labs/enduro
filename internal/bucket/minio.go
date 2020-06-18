package bucket

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

type minioBucket struct {
	sess     *session.Session
	bucket   string
	*commonBucketImpl
}

var _ Bucket = (*minioBucket)(nil)

func NewMinioBucket(ctx context.Context, config *MinioConfig) (*minioBucket, error) {
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

	return &minioBucket{
		sess:     sess,
		bucket:   config.Bucket,
		commonBucketImpl: &commonBucketImpl{
			name:             config.Name,
			retentionPeriod:  config.RetentionPeriod,
			stripTopLevelDir: config.StripTopLevelDir,
		},
	}, nil
}

func (b *minioBucket) OpenBucket(ctx context.Context) (*blob.Bucket, error) {
	return s3blob.OpenBucket(ctx, b.sess, b.bucket, nil)
}
