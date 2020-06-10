package batch

import (
	"context"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	"github.com/go-logr/logr"
	cadenceclient "go.uber.org/cadence/client"
)

//go:generate mockgen  -destination=./fake/mock_batch.go -package=fake github.com/artefactual-labs/enduro/internal/batch Service

type Service interface {
	Submit(context.Context, *goabatch.SubmitPayload) (res *goabatch.BatchResult, err error)
	Status(context.Context) (res *goabatch.BatchStatusResult, err error)
}

type batchImpl struct {
	logger logr.Logger
	cc     cadenceclient.Client
}

var _ Service = (*batchImpl)(nil)

func NewService(logger logr.Logger, cc cadenceclient.Client) *batchImpl {
	return &batchImpl{
		logger: logger,
		cc:     cc,
	}
}

func (s *batchImpl) Submit(ctx context.Context, payload *goabatch.SubmitPayload) (*goabatch.BatchResult, error) {
	// business logic
	// - implicit service dependencies, e.g. s.cc, s.logger...
	// - inputs (payload)
	// - outputs (*goabatch.BatchResult)
	// - errors (error), see e.g. `goacollection.MakeNotAvailable`
	return nil, nil
}

func (s *batchImpl) Status(ctx context.Context) (*goabatch.BatchStatusResult, error) {
	// business logic
	// s.cc
	return nil, nil
}
