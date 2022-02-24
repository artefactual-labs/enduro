package pipeline

import (
	"errors"
	"sync"

	"github.com/go-logr/logr"
)

var ErrUnknownPipeline = errors.New("unknown pipeline")

// Registry is a collection of known pipelines.
type Registry struct {
	pipelines map[string]*Pipeline
	mu        sync.Mutex
}

func NewPipelineRegistry(logger logr.Logger, configs []Config) (*Registry, error) {
	var err error
	pipelines := map[string]*Pipeline{}
	for _, config := range configs {
		logger := logger.WithValues("pipeline", config.Name)
		pipelines[config.Name], err = NewPipeline(logger, config)
		if err != nil {
			logger.Error(err, "Error connecting to pipeline", "name", config.Name)
		}
	}
	return &Registry{
		pipelines: pipelines,
	}, nil
}

func (r *Registry) List() []*Pipeline {
	r.mu.Lock()
	defer r.mu.Unlock()

	pipelines := make([]*Pipeline, 0, len(r.pipelines))
	for _, p := range r.pipelines {
		p := p
		pipelines = append(pipelines, p)
	}

	return pipelines
}

func (r *Registry) ByName(name string) (*Pipeline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pipeline, ok := r.pipelines[name]
	if !ok {
		return nil, ErrUnknownPipeline
	}

	return pipeline, nil
}

func (r *Registry) ByID(id string) (*Pipeline, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range r.pipelines {
		if p.ID == id {
			return p, nil
		}
	}

	return nil, ErrUnknownPipeline
}

func (r *Registry) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var idx int
	names := make([]string, len(r.pipelines))
	for name := range r.pipelines {
		names[idx] = name
		idx++
	}

	return names
}
