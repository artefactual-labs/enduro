package activities

import (
	"context"
	"errors"
	"fmt"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/publisher"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

// PublishTransferActivity publishes a local transfer to the pipeline location.
type PublishTransferActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewPublishTransferActivity(pipelineRegistry *pipeline.Registry) *PublishTransferActivity {
	return &PublishTransferActivity{pipelineRegistry: pipelineRegistry}
}

type PublishTransferActivityParams struct {
	PipelineName string
	FullPath     string
	RelPath      string
}

type PublishTransferActivityResult = publisher.PublishedTransfer

const PublishTransferLocalPathMissingErrorType = "PublishTransferLocalPathMissing"

func (a *PublishTransferActivity) Execute(ctx context.Context, params *PublishTransferActivityParams) (*PublishTransferActivityResult, error) {
	if params == nil {
		return nil, temporal.NewNonRetryableError(errors.New("error processing parameters: missing"))
	}

	pub, err := a.publisher(ctx, params.PipelineName)
	if err != nil {
		return nil, err
	}

	res, err := pub.Publish(ctx, params.FullPath, params.RelPath)
	if err != nil {
		return nil, publisherError(err)
	}

	return res, nil
}

func (a *PublishTransferActivity) publisher(ctx context.Context, pipelineName string) (publisher.Publisher, error) {
	p, err := a.pipelineRegistry.ByName(pipelineName)
	if err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}

	pub, err := publisher.New(
		p.Config().TransferPublisher,
		publisher.WithProgress(func(progress publisher.Progress) {
			temporalsdk_activity.RecordHeartbeat(ctx, fmt.Sprintf("Uploaded %d bytes from %s.", progress.Bytes, progress.LocalPath))
		}),
	)
	if err != nil {
		return nil, publisherError(err)
	}

	return pub, nil
}

func publisherError(err error) error {
	if publisher.IsLocalTransferMissing(err) {
		return temporalsdk_temporal.NewNonRetryableApplicationError(
			err.Error(),
			PublishTransferLocalPathMissingErrorType,
			nil,
			nil,
		)
	}
	if publisher.IsNonRetryable(err) {
		return temporal.NewNonRetryableError(err)
	}
	return err
}
