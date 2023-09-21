package manager

import (
	"fmt"
	"strings"

	"github.com/artefactual-labs/enduro/internal/collection"
)

// Manager carries workflow and activity dependencies.
type Manager struct {
	Collection collection.Service
	Hooks      map[string]map[string]interface{}
}

// NewManager returns a pointer to a new Manager.
func NewManager(colsvc collection.Service, hooks map[string]map[string]interface{}) *Manager {
	return &Manager{
		Collection: colsvc,
		Hooks:      hooks,
	}
}

func HookAttr(hooks map[string]map[string]interface{}, hook string, attr string) (interface{}, error) {
	hook = strings.ToLower(hook)
	attr = strings.ToLower(attr)

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
