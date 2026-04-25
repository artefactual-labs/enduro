package publisher

import (
	"context"
	"errors"
	"fmt"
)

type Publisher interface {
	Publish(ctx context.Context, localPath, relPath string) (*PublishedTransfer, error)
	Delete(ctx context.Context, remotePath string) error
}

type PublishedTransfer struct {
	RelPath    string
	RemotePath string
}

type LocalTransferMissingError struct {
	Path string
	err  error
}

func (e LocalTransferMissingError) Error() string {
	return fmt.Sprintf("local transfer %q is not available: %v", e.Path, e.err)
}

func (e LocalTransferMissingError) Unwrap() error {
	return e.err
}

func IsLocalTransferMissing(err error) bool {
	var missingErr LocalTransferMissingError
	return errors.As(err, &missingErr)
}

type Progress struct {
	LocalPath string
	Bytes     int64
}

type Option func(*sftpPublisher)

func WithProgress(fn func(Progress)) Option {
	return func(p *sftpPublisher) {
		p.progress = fn
	}
}

func New(cfg Config, opts ...Option) (Publisher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nonRetryable(err)
	}

	switch cfg.Type {
	case "sftp":
		p := &sftpPublisher{cfg: cfg}
		for _, opt := range opts {
			opt(p)
		}
		return p, nil
	case "":
		return noopPublisher{}, nil
	default:
		return nil, nonRetryable(fmt.Errorf("unsupported transfer publisher type %q", cfg.Type))
	}
}

type noopPublisher struct{}

func (noopPublisher) Publish(ctx context.Context, localPath, relPath string) (*PublishedTransfer, error) {
	return &PublishedTransfer{RelPath: relPath}, nil
}

func (noopPublisher) Delete(ctx context.Context, remotePath string) error {
	return nil
}

type NonRetryableError struct {
	err error
}

func (e NonRetryableError) Error() string {
	return e.err.Error()
}

func (e NonRetryableError) Unwrap() error {
	return e.err
}

func IsNonRetryable(err error) bool {
	var nonRetryableErr NonRetryableError
	return errors.As(err, &nonRetryableErr)
}

func nonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return NonRetryableError{err: err}
}
