package aipstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

type Config struct {
	Region    string
	Endpoint  string
	PathStyle bool
	Profile   string
	Key       string
	Secret    string
	Token     string
	Bucket    string
}

type Service interface {
	// UploadAIP stores a compressed AIP given its path.
	UploadAIP(ctx context.Context, path string) error

	// Close releases associated resources.
	Close() error
}

type serviceImpl struct {
	bucket *blob.Bucket
}

var _ Service = (*serviceImpl)(nil)

func NewService(config *Config) (*serviceImpl, error) {
	s := &serviceImpl{}

	var err error
	s.bucket, err = s.openBucket(config)
	if err != nil {
		return nil, fmt.Errorf("error opening bucket: %v", err)
	}

	return s, nil
}

func (s serviceImpl) openBucket(config *Config) (*blob.Bucket, error) {
	sessOpts := session.Options{}
	sessOpts.Config.WithRegion(config.Region)
	sessOpts.Config.WithEndpoint(config.Endpoint)
	sessOpts.Config.WithS3ForcePathStyle(config.PathStyle)
	sessOpts.Config.WithCredentials(
		credentials.NewStaticCredentials(
			config.Key, config.Secret, config.Token,
		),
	)
	sess, err := session.NewSessionWithOptions(sessOpts)
	if err != nil {
		return nil, err
	}

	return s3blob.OpenBucket(context.Background(), sess, config.Bucket, nil)
}

func (s *serviceImpl) UploadAIP(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	name := filepath.Base(path)
	w, err := s.bucket.NewWriter(ctx, name, &blob.WriterOptions{})
	if err != nil {
		return err
	}

	// TODO: Does this return when the context is canceled?
	_, copyErr := io.Copy(w, f)
	closeErr := w.Close()

	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return nil
}

func (s *serviceImpl) Close() error {
	return s.bucket.Close()
}

func SetBucket(s *serviceImpl, b *blob.Bucket) {
	s.bucket = b
}
