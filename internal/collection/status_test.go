package collection

import (
	"encoding/json"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		str string
		val Status
	}{
		{
			str: "new",
			val: StatusNew,
		},
		{
			str: "in progress",
			val: StatusInProgress,
		},
		{
			str: "done",
			val: StatusDone,
		},
		{
			str: "error",
			val: StatusError,
		},
		{
			str: "queued",
			val: StatusQueued,
		},
		{
			str: "abandoned",
			val: StatusAbandoned,
		},
		{
			str: "pending",
			val: StatusPending,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("Status_%s", tc.str), func(t *testing.T) {
			s := NewStatus(tc.str)
			assert.Assert(t, s != StatusUnknown)
			assert.Equal(t, s, tc.val)

			assert.Equal(t, s.String(), tc.str)

			b, err := json.Marshal(s)
			assert.NilError(t, err)
			assert.DeepEqual(t, b, []byte("\""+tc.str+"\""))

			json.Unmarshal([]byte("\""+tc.str+"\""), &s)
			assert.Assert(t, s != StatusUnknown)
			assert.Equal(t, s, tc.val)
		})
	}
}

func TestStatusUnknown(t *testing.T) {
	s := NewStatus("?")

	assert.Equal(t, s, StatusUnknown)
	assert.Equal(t, s.String(), StatusUnknown.String())
}
