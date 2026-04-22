package activities

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.artefactual.dev/amclient"
	"gotest.tools/v3/assert"
)

func TestHidePackageActivity(t *testing.T) {
	t.Run("IgnoresConflictInRecoveryMode", func(t *testing.T) {
		tests := []struct {
			name     string
			unitType string
			path     string
		}{
			{name: "transfer", unitType: "transfer", path: "/api/transfer/transfer-id/delete/"},
			{name: "ingest", unitType: "ingest", path: "/api/ingest/ingest-id/delete/"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				registry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/api/v2beta/package/":
						w.WriteHeader(http.StatusNotImplemented)
						return
					case tc.path:
						w.WriteHeader(http.StatusConflict)
						return
					default:
						http.NotFound(w, r)
					}
				})

				activity := NewHidePackageActivity(registry)
				err := activity.Execute(context.Background(), tc.unitType+"-id", tc.unitType, "am", true)
				assert.NilError(t, err)
			})
		}
	})

	t.Run("ReturnsConflictOutsideRecoveryMode", func(t *testing.T) {
		registry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v2beta/package/":
				w.WriteHeader(http.StatusNotImplemented)
			case "/api/transfer/transfer-id/delete/":
				w.WriteHeader(http.StatusConflict)
			default:
				http.NotFound(w, r)
			}
		})

		activity := NewHidePackageActivity(registry)
		err := activity.Execute(context.Background(), "transfer-id", "transfer", "am", false)
		assert.ErrorContains(t, err, "error hiding transfer")
	})
}

func TestHideConflict(t *testing.T) {
	assert.Assert(t, hideConflict(&amclient.ErrorResponse{Response: &http.Response{StatusCode: http.StatusConflict}}))
	assert.Assert(t, !hideConflict(&amclient.ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadRequest}}))
	assert.Assert(t, !hideConflict(errors.New("boom")))
}
