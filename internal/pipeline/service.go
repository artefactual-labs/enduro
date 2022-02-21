package pipeline

import (
	"context"

	"github.com/go-logr/logr"

	goapipeline "github.com/artefactual-labs/enduro/internal/api/gen/pipeline"
)

type Service interface {
	List(context.Context, *goapipeline.ListPayload) ([]*goapipeline.EnduroStoredPipeline, error)
	Show(context.Context, *goapipeline.ShowPayload) (*goapipeline.EnduroStoredPipeline, error)
	Processing(context.Context, *goapipeline.ProcessingPayload) ([]string, error)
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
		size, cur := p.Capacity()
		r := &goapipeline.EnduroStoredPipeline{
			Name:     c.Name,
			Capacity: &size,
			Current:  &cur,
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
	size, cur := pipeline.Capacity()

	return &goapipeline.EnduroStoredPipeline{
		ID:       &pipeline.ID,
		Name:     c.Name,
		Capacity: &size,
		Current:  &cur,
	}, nil
}

func (w *pipelineImpl) Processing(ctx context.Context, payload *goapipeline.ProcessingPayload) ([]string, error) {
	pipeline, err := w.registry.ByID(payload.ID)
	if err != nil {
		return nil, &goapipeline.PipelineNotFound{Message: "not_found", ID: payload.ID}
	}

	amc := pipeline.Client()

	var ret []string
	ret, _, err = amc.ProcessingConfig.List(ctx)

	return ret, err
}
