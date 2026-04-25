package activities

import (
	"context"
	"errors"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/publisher"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

// CleanUpPublishedTransferActivity deletes a published transfer from storage.
type CleanUpPublishedTransferActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewCleanUpPublishedTransferActivity(pipelineRegistry *pipeline.Registry) *CleanUpPublishedTransferActivity {
	return &CleanUpPublishedTransferActivity{pipelineRegistry: pipelineRegistry}
}

type CleanUpPublishedTransferActivityParams struct {
	PipelineName string
	RemotePath   string
}

func (a *CleanUpPublishedTransferActivity) Execute(ctx context.Context, params *CleanUpPublishedTransferActivityParams) error {
	if params == nil {
		return temporal.NewNonRetryableError(errors.New("error processing parameters: missing"))
	}
	if params.RemotePath == "" {
		return nil
	}

	pub, err := a.publisher(ctx, params.PipelineName)
	if err != nil {
		return err
	}

	if err := pub.Delete(ctx, params.RemotePath); err != nil {
		return publisherError(err)
	}

	return nil
}

func (a *CleanUpPublishedTransferActivity) publisher(ctx context.Context, pipelineName string) (publisher.Publisher, error) {
	p, err := a.pipelineRegistry.ByName(pipelineName)
	if err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}

	pub, err := publisher.New(p.Config().TransferPublisher)
	if err != nil {
		return nil, publisherError(err)
	}

	return pub, nil
}
