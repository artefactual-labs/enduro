package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/go-logr/logr/funcr"
	goahttp "goa.design/goa/v3/http"
	goahttpmwr "goa.design/goa/v3/http/middleware"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/api/gen/collection"
	collectionsvr "github.com/artefactual-labs/enduro/internal/api/gen/http/collection/server"
)

func TestMonitorHTTPError(t *testing.T) {
	t.Parallel()

	for _, started := range []bool{false, true} {
		name := "before streaming"
		if started {
			name = "after streaming"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var logs strings.Builder
			logger := funcr.New(func(_, message string) { logs.WriteString(message) }, funcr.Options{})
			endpoint := func(ctx context.Context, input any) (any, error) {
				if started {
					stream := input.(*collection.MonitorEndpointInput).Stream
					err := stream.SendWithContext(ctx, &collection.EnduroMonitorUpdate{
						Type:      "hello",
						Timestamp: "2026-10-22T12:00:00Z",
					})
					if err != nil {
						return nil, err
					}
				}
				return nil, errors.New("monitor failed")
			}
			handler := collectionsvr.NewMonitorHandler(endpoint, goahttp.NewMuxer(),
				goahttp.RequestDecoder, goahttp.ResponseEncoder, errorHandler(logger, "Collection error."), nil)
			request := httptest.NewRequest(http.MethodGet, "/collection/monitor", nil)
			request.Header.Set("X-Request-Id", "request-42")
			response := httptest.NewRecorder()

			// Match the request-ID middleware still used by HTTPServer.
			goahttpmwr.RequestID(goahttpmwr.UseXRequestIDHeaderOption(true))(handler).ServeHTTP(response, request) //nolint:staticcheck

			if !started {
				assert.Equal(t, response.Code, http.StatusInternalServerError)
				assert.Assert(t, strings.Contains(response.Body.String(), "monitor failed"))
				return
			}
			assert.Equal(t, response.Code, http.StatusOK)
			assert.Equal(t, response.Header().Get("Content-Type"), "text/event-stream")
			body := response.Body.String()
			assert.Assert(t, strings.HasPrefix(body, "data: "))
			assert.Assert(t, strings.HasSuffix(body, "\n\n"))
			var event map[string]any
			assert.NilError(t, json.Unmarshal([]byte(strings.TrimPrefix(body, "data: ")), &event))
			assert.DeepEqual(t, event, map[string]any{
				"id":        float64(0),
				"timestamp": "2026-10-22T12:00:00Z",
				"type":      "hello",
			})
			assert.Assert(t, strings.Contains(logs.String(), "monitor failed"))
			assert.Assert(t, strings.Contains(logs.String(), "request-42"))
			assert.Assert(t, strings.Contains(logs.String(), "Collection error."))
		})
	}
}

func TestDownloadHTTPReadError(t *testing.T) {
	t.Parallel()

	const content = "partial archive"
	var logs strings.Builder
	logger := funcr.New(func(_, message string) { logs.WriteString(message) }, funcr.Options{})
	endpoint := func(context.Context, any) (any, error) {
		return &collection.DownloadResponseData{
			Result: &collection.DownloadResult{
				ContentType:        "application/octet-stream",
				ContentLength:      100,
				ContentDisposition: "attachment; filename=archive.tar",
			},
			Body: io.NopCloser(io.MultiReader(strings.NewReader(content), iotest.ErrReader(io.ErrUnexpectedEOF))),
		}, nil
	}
	mux := goahttp.NewMuxer()
	handler := collectionsvr.NewDownloadHandler(endpoint, mux,
		goahttp.RequestDecoder, goahttp.ResponseEncoder, errorHandler(logger, "Collection error."), nil)
	collectionsvr.MountDownloadHandler(mux, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/collection/42/download")
	assert.NilError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	assert.Equal(t, response.StatusCode, http.StatusOK)
	assert.Equal(t, string(body), content)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}
