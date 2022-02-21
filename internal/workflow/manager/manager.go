package manager

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/watcher"
)

// Manager carries workflow and activity dependencies.
type Manager struct {
	Logger     logr.Logger
	Collection collection.Service
	Watcher    watcher.Service
	Pipelines  *pipeline.Registry
	Hooks      map[string]map[string]interface{}
}

// NewManager returns a pointer to a new Manager.
func NewManager(logger logr.Logger, colsvc collection.Service, wsvc watcher.Service, pipelines *pipeline.Registry, hooks map[string]map[string]interface{}) *Manager {
	return &Manager{
		Logger:     logger,
		Collection: colsvc,
		Watcher:    wsvc,
		Pipelines:  pipelines,
		Hooks:      hooks,
	}
}

func HookAttr(hooks map[string]map[string]interface{}, hook string, attr string) (interface{}, error) {
	configMap, ok := hooks[hook]
	if !ok {
		return "", fmt.Errorf("hook %q not found", hook)
	}

	value, ok := configMap[attr]
	if !ok {
		return "", fmt.Errorf("attr %q not found", attr)
	}

	return value, nil
}

func HookAttrString(hooks map[string]map[string]interface{}, hook string, attr string) (string, error) {
	accessor := fmt.Sprintf("%s:%s", hook, attr)

	value, err := HookAttr(hooks, hook, attr)
	if err != nil {
		return "", fmt.Errorf("error accessing %q", accessor)
	}

	v, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("error accessing %q: not a string", accessor)
	}

	return v, nil
}

func HookAttrBool(hooks map[string]map[string]interface{}, hook string, attr string) (bool, error) {
	accessor := fmt.Sprintf("%s:%s", hook, attr)

	value, err := HookAttr(hooks, hook, attr)
	if err != nil {
		return false, fmt.Errorf("error accessing %q", accessor)
	}

	v, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("error accessing %q: not a boolean", accessor)
	}

	return v, nil
}
