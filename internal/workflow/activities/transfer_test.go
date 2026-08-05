package activities

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/temporal"
)

func TestTransferActivityIdempotencyVersionGate(t *testing.T) {
	tests := map[string]struct {
		version        string
		serverInfoCode int
		wantKey        bool
	}{
		"Older server": {
			version:        "1.18.9",
			serverInfoCode: http.StatusNotImplemented,
		},
		"Minimum supported server": {
			version:        "1.19.0",
			serverInfoCode: http.StatusNotImplemented,
			wantKey:        true,
		},
		"Newer server": {
			version:        "2.0.0",
			serverInfoCode: http.StatusNotImplemented,
			wantKey:        true,
		},
		"Missing version": {
			serverInfoCode: http.StatusNotImplemented,
		},
		"Malformed version": {
			version:        "development",
			serverInfoCode: http.StatusNotImplemented,
		},
		"Failed discovery": {
			serverInfoCode: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				getRequests  int
				postRequests int
				gotKey       string
			)
			pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					getRequests++
					if tc.version != "" {
						w.Header().Set("X-Archivematica-Version", tc.version)
					}
					w.WriteHeader(tc.serverInfoCode)
				case http.MethodPost:
					postRequests++
					gotKey = r.Header.Get("Idempotency-Key")
					w.Header().Set("X-Archivematica-Version", "1.19.0")
					w.Header().Set("X-Archivematica-ID", "pipeline-id")
					w.WriteHeader(http.StatusAccepted)
					fmt.Fprint(w, `{"id":"transfer-id"}`)
				default:
					t.Fatalf("unexpected request method: %s", r.Method)
				}
			})

			result, err := executeTransferActivity(t, NewTransferActivity(pipelineRegistry))

			assert.NilError(t, err)
			assert.Equal(t, result.TransferID, "transfer-id")
			assert.Equal(t, result.PipelineID, "pipeline-id")
			assert.Equal(t, getRequests, 1)
			assert.Equal(t, postRequests, 1)
			if tc.wantKey {
				assert.Assert(t, strings.HasPrefix(gotKey, "enduro-transfer-"))
				return
			}
			assert.Equal(t, gotKey, "")
		})
	}
}

func TestTransferActivityChecksVersionEveryAttempt(t *testing.T) {
	var (
		serverInfoRequests int
		gotKeys            []string
	)
	pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			serverInfoRequests++
			version := "1.18.9"
			if serverInfoRequests > 1 {
				version = "1.19.0"
			}
			w.Header().Set("X-Archivematica-Version", version)
			w.WriteHeader(http.StatusNotImplemented)
		case http.MethodPost:
			gotKeys = append(gotKeys, r.Header.Get("Idempotency-Key"))
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"id":"transfer-id"}`)
		default:
			t.Fatalf("unexpected request method: %s", r.Method)
		}
	})
	activity := NewTransferActivity(pipelineRegistry)

	for range 2 {
		_, err := executeTransferActivity(t, activity)
		assert.NilError(t, err)
	}

	assert.Equal(t, serverInfoRequests, 2)
	assert.Equal(t, len(gotKeys), 2)
	assert.Equal(t, gotKeys[0], "")
	assert.Assert(t, strings.HasPrefix(gotKeys[1], "enduro-transfer-"))
}

func TestTransferActivityIdempotencyErrors(t *testing.T) {
	tests := map[string]struct {
		statusCode   int
		nonRetryable bool
	}{
		"Request in progress remains retryable": {
			statusCode: http.StatusConflict,
		},
		"Changed request is not retryable": {
			statusCode:   http.StatusUnprocessableEntity,
			nonRetryable: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.Header().Set("X-Archivematica-Version", "1.19.0")
					w.WriteHeader(http.StatusNotImplemented)
					return
				}
				w.WriteHeader(tc.statusCode)
			})

			result, err := executeTransferActivity(t, NewTransferActivity(pipelineRegistry))

			assert.Assert(t, result == nil)
			assert.Assert(t, err != nil)
			assert.Equal(t, temporal.NonRetryableError(err), tc.nonRetryable)
			assert.ErrorContains(t, err, fmt.Sprintf("%d", tc.statusCode))
		})
	}
}

func TestTransferIdempotencyKey(t *testing.T) {
	info := temporalsdk_activity.Info{}
	assert.Equal(t, transferIdempotencyKey(info), "")

	info.WorkflowExecution.RunID = "run-id"
	assert.Equal(t, transferIdempotencyKey(info), "enduro-transfer-run-id")
	assert.Equal(t, transferIdempotencyKey(info), transferIdempotencyKey(info))

	otherInfo := info
	otherInfo.WorkflowExecution.RunID = "other-run-id"
	assert.Assert(t, transferIdempotencyKey(info) != transferIdempotencyKey(otherInfo))
}

func executeTransferActivity(t *testing.T, activity *TransferActivity) (*TransferActivityResponse, error) {
	t.Helper()

	s := temporalsdk_testsuite.WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activity.Execute)

	future, err := env.ExecuteActivity(activity.Execute, &TransferActivityParams{
		PipelineName:       "am",
		TransferLocationID: "location-id",
		RelPath:            "transfer/path",
		Name:               "transfer",
		ProcessingConfig:   "automated",
		TransferType:       "standard",
		Accession:          "accession",
	})
	if err != nil {
		return nil, err
	}

	result := &TransferActivityResponse{}
	if err := future.Get(result); err != nil {
		return nil, err
	}

	return result, nil
}
