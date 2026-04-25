package publisher

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestCleanRelPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input       string
		want        string
		errContains string
	}{
		"Cleans relative paths": {
			input: "batch/../transfer",
			want:  "transfer",
		},
		"Converts platform separators": {
			input: `batch\transfer`,
			want:  "batch/transfer",
		},
		"Rejects empty paths": {
			input:       "",
			errContains: "non-empty transfer path",
		},
		"Rejects absolute paths": {
			input:       "/transfer",
			errContains: "relative transfer path",
		},
		"Rejects parent traversal": {
			input:       "../transfer",
			errContains: "within the transfer source",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := cleanRelPath(tc.input)
			if tc.errContains == "" {
				assert.NilError(t, err)
				assert.Equal(t, got, tc.want)
				return
			}

			assert.ErrorContains(t, err, tc.errContains)
		})
	}
}
