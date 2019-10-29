package amclient

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIngest_Hide(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/ingest/52dd0c01-e803-423a-be5f-b592b5d5d61c/delete/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
	"removed": true
}`)
	})

	payload, _, err := client.Ingest.Hide(ctx, "52dd0c01-e803-423a-be5f-b592b5d5d61c")
	if err != nil {
		t.Errorf("Ingest.Hide() returned error: %v", err)
	}

	assert.NoError(t, err)
	assert.True(t, payload.Removed)
}
