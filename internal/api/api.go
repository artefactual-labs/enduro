/*
Package api contains the API server.

HTTP is the only transport supported at the moment.

The design package is the Goa design package while the gen package contains all
the generated code produced with goa gen.
*/
package api

import (
	"context"
	"net/http"
	"os"
	"time"

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

	"github.com/go-logr/logr"
	goahttp "goa.design/goa/v3/http"
	goahttpmwr "goa.design/goa/v3/http/middleware"
	"goa.design/goa/v3/middleware"
)

func HTTPServer(
	logger logr.Logger, config *Config,
	pipesvc intpipe.Service,
	batchsvc intbatch.Service,
	colsvc intcol.Service,
) *http.Server {
	var dec = goahttp.RequestDecoder
	var enc = goahttp.ResponseEncoder
	var mux goahttp.Muxer = goahttp.NewMuxer()

	// Pipeline service.
	var pipelineEndpoints *pipeline.Endpoints = pipeline.NewEndpoints(pipesvc)
	var pipelineErrorHandler = errorHandler(logger, "Pipeline error.")
	var pipelineServer *pipelinesvr.Server = pipelinesvr.New(pipelineEndpoints, mux, dec, enc, pipelineErrorHandler, nil)
	pipelinesvr.Mount(mux, pipelineServer)

	// Batch service.
	var batchEndpoints *batch.Endpoints = batch.NewEndpoints(batchsvc)
	var batchErrorHandler = errorHandler(logger, "Batch error.")
	var batchServer *batchsvr.Server = batchsvr.New(batchEndpoints, mux, dec, enc, batchErrorHandler, nil)
	batchsvr.Mount(mux, batchServer)

	// Collection service.
	var collectionEndpoints *collection.Endpoints = collection.NewEndpoints(colsvc.Goa())
	var collectionErrorHandler = errorHandler(logger, "Collection error.")
	var collectionServer *collectionsvr.Server = collectionsvr.New(collectionEndpoints, mux, dec, enc, collectionErrorHandler, nil)
	// Intercept request in Download endpoint so we can serve the file directly.
	collectionServer.Download = colsvc.HTTPDownload(mux, dec)
	collectionsvr.Mount(mux, collectionServer)

	// Swagger service.
	var swaggerService *swaggersvr.Server = swaggersvr.New(nil, nil, nil, nil, nil, nil)
	swaggersvr.Mount(mux, swaggerService)

	// Web handler.
	web, err := ui.Handler()
	if err != nil {
		logger.Error(err, "This build does not embed the ui package!")
	} else {
		mux.Handle("GET", "/", web.ServeHTTP)
		mux.Handle("GET", "/{*filename}", web.ServeHTTP)
	}

	// Global middlewares.
	var handler http.Handler = mux
	handler = goahttpmwr.RequestID()(handler)
	handler = versionHeaderMiddleware(config.AppVersion)(handler)
	if config.Debug {
		handler = goahttpmwr.Log(loggerAdapter(logger))(handler)
		handler = goahttpmwr.Debug(mux, os.Stdout)(handler)
	}

	return &http.Server{
		Addr:         config.Listen,
		Handler:      handler,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 10,
		IdleTimeout:  time.Second * 120,
	}
}

// errorHandler returns a function that writes and logs the given error.
// The function also writes and logs the error unique ID so that it's possible
// to correlate.
func errorHandler(logger logr.Logger, msg string) func(context.Context, http.ResponseWriter, error) {
	return func(ctx context.Context, w http.ResponseWriter, err error) {
		id := ctx.Value(middleware.RequestIDKey).(string)
		_, _ = w.Write([]byte("[" + id + "] encoding: " + err.Error()))
		logger.Error(err, "Package service error.")
	}
}

func versionHeaderMiddleware(version string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Enduro-Version", version)
			h.ServeHTTP(w, r)
		})
	}
}
