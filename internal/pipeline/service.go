package pipeline

import (
	"context"

	goapipeline "github.com/artefactual-labs/enduro/internal/api/gen/pipeline"
	"github.com/go-logr/logr"
)

//go:generate mockgen  -destination=./fake/mock_pipeline.go -package=fake github.com/artefactual-labs/enduro/internal/pipeline Service

type Service interface {
	List(context.Context, *goapipeline.ListPayload) (res []*goapipeline.EnduroStoredPipeline, err error)
	Show(context.Context, *goapipeline.ShowPayload) (res *goapipeline.EnduroStoredPipeline, err error)
}

type pipelineImpl struct {
	logger   logr.Logger
	registry *Registry
}

var _ Service = (*pipelineImpl)(nil)

func NewService(logger logr.Logger, registry *Registry) *pipelineImpl {
	return &pipelineImpl{
		logger:   logger,
		registry: registry,
	}
}

func (w *pipelineImpl) List(ctx context.Context, payload *goapipeline.ListPayload) ([]*goapipeline.EnduroStoredPipeline, error) {
	pipelines := w.registry.List()
	results := make([]*goapipeline.EnduroStoredPipeline, 0, len(pipelines))

	for _, p := range pipelines {
		c := p.Config()
		r := &goapipeline.EnduroStoredPipeline{
			Name: c.Name,
		}
		if p.ID != "" {
			r.ID = &p.ID
		}
		results = append(results, r)
	}

	return results, nil
}

func (w *pipelineImpl) Show(ctx context.Context, payload *goapipeline.ShowPayload) (*goapipeline.EnduroStoredPipeline, error) {
	pipeline, err := w.registry.ByID(payload.ID)
	if err != nil {
		return nil, err
	}

	c := pipeline.Config()

	return &goapipeline.EnduroStoredPipeline{
		ID:   &pipeline.ID,
		Name: c.Name,
	}, nil
}
