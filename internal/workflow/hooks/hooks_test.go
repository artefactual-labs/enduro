package hooks

import (
	"testing"

	"gotest.tools/v3/assert"
)

var sampleHooks = map[string]map[string]any{
	"hari": {
		"baseurl":  "https://192.168.1.50:8080/api",
		"disabled": false,
		"mock":     false,
	},
	"prod": {
		"disabled":    false,
		"receiptpath": "./hack/production-system-interface",
	},
}

func TestHookAttrString(t *testing.T) {
	value, err := HookAttrString(sampleHooks, "hari", "baseURL")

	assert.NilError(t, err)
	assert.Equal(t, value, "https://192.168.1.50:8080/api")
}

func TestHookAttrBool(t *testing.T) {
	value, err := HookAttrBool(sampleHooks, "hari", "disabled")

	assert.NilError(t, err)
	assert.Equal(t, value, false)
}
