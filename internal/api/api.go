package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/artefactual-labs/enduro/internal/api/gen/collection"
	collectionsvr "github.com/artefactual-labs/enduro/internal/api/gen/http/collection/server"
	swaggersvr "github.com/artefactual-labs/enduro/internal/api/gen/http/swagger/server"
	"github.com/artefactual-labs/enduro/ui"

	"github.com/go-logr/logr"
	"goa.design/goa/middleware"
	goahttp "goa.design/goa/v3/http"
	goahttpmwr "goa.design/goa/v3/http/middleware"
)

func HTTPServer(logger logr.Logger, config *Config, colsvc collection.Service) *http.Server {
	var dec = goahttp.RequestDecoder
	var enc = goahttp.ResponseEncoder
	var mux goahttp.Muxer = goahttp.NewMuxer()

	// Collection service.
	var collectionService collection.Service = colsvc
	var collectionEndpoints *collection.Endpoints = collection.NewEndpoints(collectionService)
	var collectionErrorHandler = errorHandler(logger, "Collection error.")
	var collectionServer *collectionsvr.Server = collectionsvr.New(collectionEndpoints, mux, dec, enc, collectionErrorHandler)
	collectionsvr.Mount(mux, collectionServer)

	// Swagger service.
	var swaggerService *swaggersvr.Server = swaggersvr.New(nil, nil, nil, nil, nil)
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
