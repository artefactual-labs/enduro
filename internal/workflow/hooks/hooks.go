package hooks

import (
	"fmt"
	"strings"
)

// Hooks carries workflow and activity dependencies.
type Hooks struct {
	Hooks map[string]map[string]any
}

// NewHooks returns a pointer to a new Hooks.
func NewHooks(hooks map[string]map[string]any) *Hooks {
	return &Hooks{
		Hooks: hooks,
	}
}

func HookAttr(hooks map[string]map[string]any, hook, attr string) (any, error) {
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

func HookAttrString(hooks map[string]map[string]any, hook, attr string) (string, error) {
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

func HookAttrBool(hooks map[string]map[string]any, hook, attr string) (bool, error) {
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
