package bucket

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Bucket interface {
	RetentionPeriod() *time.Duration
	StripTopLevelDir() bool
	fmt.Stringer
}

type commonBucketImpl struct {
	name             string
	retentionPeriod  *time.Duration
	stripTopLevelDir bool
}

func (b *commonBucketImpl) String() string {
	return b.name
}

func (b *commonBucketImpl) RetentionPeriod() *time.Duration {
	return b.retentionPeriod
}

func (b *commonBucketImpl) StripTopLevelDir() bool {
	return b.stripTopLevelDir
}

type Service interface {
	Buckets() []Bucket
}

type serviceImpl struct {
	buckets map[string]Bucket
	mu       sync.RWMutex
}

var _ Service = (*serviceImpl)(nil)

func New(ctx context.Context, c *Config) (*serviceImpl, error) {
	var buckets = map[string]Bucket{}

	for _, item := range c.Minio {
		item := item
		b, err := NewMinioBucket(ctx, item)
		if err != nil {
			return nil, err
		}

		buckets[item.Name] = b
	}

	for _, item := range c.Filesystem {
		item := item
		b, err := NewFilesystemBucket(ctx, item)
		if err != nil {
			return nil, err
		}

		buckets[item.Name] = b
	}

	if len(buckets) == 0 {
		return nil, errors.New("there are not buckets configured")
	}

	return &serviceImpl{buckets: buckets}, nil
}

func (svc *serviceImpl) Buckets() []Bucket {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	var buckets = []Bucket{}
	for _, item := range svc.buckets {
		item := item
		buckets = append(buckets, item)
	}

	return buckets
}

func (svc *serviceImpl) bucket(name string) (Bucket, error) {
	svc.mu.RLock()
	defer svc.mu.RUnlock()

	b, ok := svc.buckets[name]
	if !ok {
		return nil, fmt.Errorf("error loading bucket: unknown bucket %s", name)
	}

	return b, nil
}
