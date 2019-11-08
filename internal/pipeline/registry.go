package pipeline

import (
	"errors"
	"sync"
)

var ErrUnknownPipeline = errors.New("unknown pipeline")

// Registry is a collection of known pipelines.
type Registry struct {
	pipelines map[string]*Pipeline
	mu        sync.Mutex
}

func NewPipelineRegistry(configs []Config) *Registry {
	pipelines := map[string]*Pipeline{}
	for _, config := range configs {
		pipelines[config.Name] = NewPipeline(&config)
	}
	return &Registry{
		pipelines: pipelines,
	}
}

func (r *Registry) Pipeline(name string) (*Pipeline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pipeline, ok := r.pipelines[name]
	if !ok {
		return nil, ErrUnknownPipeline
	}

	return pipeline, nil
}
