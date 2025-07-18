/*
Package api contains the API server.

HTTP is the only transport supported at the moment.

The design package is the Goa design package while the gen package contains all
the generated code produced with goa gen.
*/
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"
	"unicode/utf8"

	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	goahttp "goa.design/goa/v3/http"
	goahttpmwr "goa.design/goa/v3/http/middleware"
	"goa.design/goa/v3/middleware"

	"github.com/artefactual-labs/enduro/internal/api/gen/batch"
	"github.com/artefactual-labs/enduro/internal/api/gen/collection"
	batchsvr "github.com/artefactual-labs/enduro/internal/api/gen/http/batch/server"
	collectionsvr "github.com/artefactual-labs/enduro/internal/api/gen/http/collection/server"
	pipelinesvr "github.com/artefactual-labs/enduro/internal/api/gen/http/pipeline/server"
	swaggersvr "github.com/artefactual-labs/enduro/internal/api/gen/http/swagger/server"
	"github.com/artefactual-labs/enduro/internal/api/gen/pipeline"
	intbatch "github.com/artefactual-labs/enduro/internal/batch"
	intcol "github.com/artefactual-labs/enduro/internal/collection"
	intpipe "github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/ui"
)

func HTTPServer(
	logger logr.Logger,
	tp trace.TracerProvider,
	config *Config,
	pipesvc intpipe.Service,
	batchsvc intbatch.Service,
	colsvc intcol.Service,
) *http.Server {
	dec := goahttp.RequestDecoder
	enc := goahttp.ResponseEncoder
	var mux goahttp.Muxer = goahttp.NewMuxer()

	websocketUpgrader := &websocket.Upgrader{
		HandshakeTimeout: time.Second,
		CheckOrigin:      sameOriginChecker(logger),
	}

	// Pipeline service.
	pipelineEndpoints := pipeline.NewEndpoints(pipesvc)
	pipelineErrorHandler := errorHandler(logger, "Pipeline error.")
	pipelineServer := pipelinesvr.New(pipelineEndpoints, mux, dec, enc, pipelineErrorHandler, nil)
	pipelinesvr.Mount(mux, pipelineServer)

	// Batch service.
	batchEndpoints := batch.NewEndpoints(batchsvc)
	batchErrorHandler := errorHandler(logger, "Batch error.")
	batchServer := batchsvr.New(batchEndpoints, mux, dec, enc, batchErrorHandler, nil)
	batchsvr.Mount(mux, batchServer)

	// Collection service.
	collectionEndpoints := collection.NewEndpoints(colsvc.Goa())
	collectionErrorHandler := errorHandler(logger, "Collection error.")
	collectionServer := collectionsvr.New(collectionEndpoints, mux, dec, enc, collectionErrorHandler, nil, websocketUpgrader, nil)
	collectionServer.Download = writeTimeout(collectionServer.Download, 0)
	collectionsvr.Mount(mux, collectionServer)

	// Swagger service.
	swaggerService := swaggersvr.New(nil, nil, nil, nil, nil, nil, nil)
	swaggersvr.Mount(mux, swaggerService)

	// Web handler.
	web := ui.SPAHandler()
	mux.Handle("GET", "/", web)
	mux.Handle("GET", "/{*filename}", web)

	// Global middlewares.
	var handler http.Handler = mux
	handler = otelhttp.NewHandler(handler, "enduro/internal/api", otelhttp.WithTracerProvider(tp))
	handler = goahttpmwr.RequestID()(handler)
	handler = versionHeaderMiddleware(config.AppVersion)(handler)
	if config.Debug {
		handler = goahttpmwr.Log(loggerAdapter(logger))(handler)
		handler = debug(mux, os.Stdout)(handler)
	}

	return &http.Server{
		Addr:         config.Listen,
		Handler:      handler,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 120,
	}
}

type errorMessage struct {
	RequestID string
	Error     error
}

// errorHandler returns a function that writes and logs the given error.
// The function also writes and logs the error unique ID so that it's possible
// to correlate.
func errorHandler(logger logr.Logger, msg string) func(context.Context, http.ResponseWriter, error) {
	return func(ctx context.Context, w http.ResponseWriter, err error) {
		reqID, ok := ctx.Value(middleware.RequestIDKey).(string)
		if !ok {
			reqID = "unknown"
		}

		// Only write the error if the connection is not hijacked.
		var ws bool
		if _, err := w.Write(nil); err == http.ErrHijacked {
			ws = true
		} else {
			_ = json.NewEncoder(w).Encode(&errorMessage{RequestID: reqID})
		}

		logger.Error(err, "Service error.", "reqID", reqID, "ws", ws)
	}
}

func sameOriginChecker(logger logr.Logger) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header["Origin"]
		if len(origin) == 0 {
			return true
		}
		u, err := url.Parse(origin[0])
		if err != nil {
			logger.V(1).Info("WebSocket client rejected (origin parse error)", "err", err)
			return false
		}
		eq := equalASCIIFold(u.Host, r.Host)
		if !eq {
			logger.V(1).Info("WebSocket client rejected (origin and host not equal)", "origin-host", u.Host, "request-host", r.Host)
		}
		return eq
	}
}

// equalASCIIFold returns true if s is equal to t with ASCII case folding as
// defined in RFC 4790.
func equalASCIIFold(s, t string) bool {
	for s != "" && t != "" {
		sr, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		tr, size := utf8.DecodeRuneInString(t)
		t = t[size:]
		if sr == tr {
			continue
		}
		if 'A' <= sr && sr <= 'Z' {
			sr = sr + 'a' - 'A'
		}
		if 'A' <= tr && tr <= 'Z' {
			tr = tr + 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return s == t
}
