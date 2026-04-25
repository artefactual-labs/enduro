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
	"os"
	"time"

	"github.com/go-logr/logr"
	"go.artefactual.dev/tools/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	goahttp "goa.design/goa/v3/http"
	goahttpmwr "goa.design/goa/v3/http/middleware"
	goamiddleware "goa.design/goa/v3/middleware"

	frontendui "github.com/artefactual-labs/enduro/frontend"
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

type UIMode int

const (
	UIModeLegacy UIMode = iota
	UIModeNuxt
)

func HTTPServer(
	logger logr.Logger,
	tp trace.TracerProvider,
	config *Config,
	pipesvc intpipe.Service,
	batchsvc intbatch.Service,
	colsvc intcol.Service,
	uiMode UIMode,
) *http.Server {
	dec := goahttp.RequestDecoder
	enc := goahttp.ResponseEncoder
	mux := goahttp.NewMuxer()
	mux.Use(otelhttp.NewMiddleware("enduro/internal/api", otelhttp.WithTracerProvider(tp)))

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
	collectionServer := collectionsvr.New(collectionEndpoints, mux, dec, enc, collectionErrorHandler, nil)
	collectionServer.Monitor = middleware.WriteTimeout(0)(collectionServer.Monitor)
	collectionServer.Download = middleware.WriteTimeout(0)(collectionServer.Download)
	collectionsvr.Mount(mux, collectionServer)

	// Swagger service.
	swaggerService := swaggersvr.New(nil, nil, nil, nil, nil, nil, nil)
	swaggersvr.Mount(mux, swaggerService)

	switch uiMode {
	case UIModeNuxt:
		nuxt := frontendui.SPAHandler("/")
		mux.Handle("GET", "/", nuxt)
		mux.Handle("GET", "/{*filename}", nuxt)
	case UIModeLegacy:
		legacy := ui.SPAHandler()
		mux.Handle("GET", "/", legacy)
		mux.Handle("GET", "/{*filename}", legacy)
	}

	// Global middlewares.
	var handler http.Handler = mux
	handler = goahttpmwr.RequestID()(handler)
	handler = corsResponseHeaderMiddleware(config.AllowedOrigins)(handler)
	handler = crossOriginProtectionMiddleware(config.AllowedOrigins)(handler)
	handler = middleware.VersionHeader("X-Enduro-Version", config.AppVersion)(handler)
	handler = securityHeadersMiddleware(config.ContentSecurityPolicy)(handler)
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
		reqID, ok := ctx.Value(goamiddleware.RequestIDKey).(string)
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
