package bucket

import (
	"context"
	"errors"
	"path/filepath"
	"fmt"
	"os"

	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
)

type filesystemBucket struct {
	ctx   context.Context
	path  string
	*commonBucketImpl
}

var _ Bucket = (*filesystemBucket)(nil)

func NewFilesystemBucket(ctx context.Context, config *FilesystemConfig) (*filesystemBucket, error) {
	stat, err := os.Stat(config.Path)
	if err != nil {
		return nil, fmt.Errorf("error looking up stat info: %w", err)
	}
	if !stat.IsDir() {
		return nil, errors.New("given path is not a directory")
	}
	abspath, err := filepath.Abs(config.Path)
	if err != nil {
		return nil, fmt.Errorf("error generating absolute path of %s: %v", config.Path, err)
	}

	return &filesystemBucket{
		ctx:   ctx,
		path:  abspath,
		commonBucketImpl: &commonBucketImpl{
			name:             config.Name,
			retentionPeriod:  config.RetentionPeriod,
			stripTopLevelDir: config.StripTopLevelDir,
		},
	}, nil

}

func (b *filesystemBucket) OpenBucket(context.Context) (*blob.Bucket, error) {
	return fileblob.OpenBucket(b.path, nil)
}
