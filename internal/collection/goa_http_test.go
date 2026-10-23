package collection

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goahttp "goa.design/goa/v3/http"
	"gotest.tools/v3/assert"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	collectionsvr "github.com/artefactual-labs/enduro/internal/api/gen/http/collection/server"
)

func TestCollectionShowHTTPWorkflowID(t *testing.T) {
	t.Parallel()

	for _, workflowID := range []string{
		"processing-workflow-04e9257e-ac59-442c-a037-7504ea5ebf3f",
		"custom-workflow-42",
		"04e9257e-ac59-442c-a037-7504ea5ebf3f",
		"",
	} {
		t.Run(workflowID, func(t *testing.T) {
			t.Parallel()

			recorder := newExecRecorderDB(t)
			recorder.row = &Collection{
				ID:         42,
				WorkflowID: workflowID,
				RunID:      "d1845cb6-a5ea-474a-9ab8-26f9bcd919f5",
				Status:     StatusInProgress,
				CreatedAt:  time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC),
			}
			svc := NewService(testLogger(), recorder.db, nil, "", nil)
			mux := goahttp.NewMuxer()
			server := collectionsvr.New(goacollection.NewEndpoints(svc.Goa()), mux,
				goahttp.RequestDecoder, goahttp.ResponseEncoder, nil, nil)
			collectionsvr.Mount(mux, server)

			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/collection/42", nil))

			assert.Equal(t, response.Code, http.StatusOK, "response: %s", response.Body.String())
			var body struct {
				ID         uint   `json:"id"`
				WorkflowID string `json:"workflow_id"`
				RunID      string `json:"run_id"`
			}
			assert.NilError(t, json.Unmarshal(response.Body.Bytes(), &body))
			assert.Equal(t, body.ID, uint(42))
			assert.Equal(t, body.WorkflowID, workflowID)
			assert.Equal(t, body.RunID, recorder.row.RunID)
		})
	}
}

func TestCollectionStatusHistoryHTTPEmpty(t *testing.T) {
	t.Parallel()

	recorder := newExecRecorderDB(t)
	recorder.row = &Collection{
		ID:         42,
		WorkflowID: "processing-workflow-42",
		Status:     StatusDone,
		CreatedAt:  time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC),
	}
	svc := NewService(testLogger(), recorder.db, nil, "", nil)
	mux := goahttp.NewMuxer()
	server := collectionsvr.New(goacollection.NewEndpoints(svc.Goa()), mux,
		goahttp.RequestDecoder, goahttp.ResponseEncoder, nil, nil)
	collectionsvr.Mount(mux, server)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/collection/42/status-history", nil))

	assert.Equal(t, response.Code, http.StatusOK, "response: %s", response.Body.String())
	var body struct {
		Availability string `json:"availability"`
		Transitions  []any  `json:"transitions"`
	}
	assert.NilError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, body.Availability, StatusHistoryUnavailable)
	assert.DeepEqual(t, body.Transitions, []any{})
}

func TestCollectionDecideHTTPInvalidOption(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{}`, `{"option":null}`, `{"option":""}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()

			recorder := newExecRecorderDB(t)
			recorder.row = &Collection{
				ID:         42,
				WorkflowID: "processing-workflow-42",
				Status:     StatusPending,
			}
			svc := NewService(testLogger(), recorder.db, nil, "", nil)
			mux := goahttp.NewMuxer()
			server := collectionsvr.New(goacollection.NewEndpoints(svc.Goa()), mux,
				goahttp.RequestDecoder, goahttp.ResponseEncoder, nil, nil)
			collectionsvr.Mount(mux, server)
			request := httptest.NewRequest(http.MethodPost, "/collection/42/decision", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			assert.Equal(t, response.Code, http.StatusBadRequest, "response: %s", response.Body.String())
			assert.Equal(t, response.Header().Get("goa-error"), "not_valid")
		})
	}
}
