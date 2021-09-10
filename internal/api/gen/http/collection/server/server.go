// Code generated by goa v3.5.1, DO NOT EDIT.
//
// collection HTTP server
//
// Command:
// $ goa-v3.4.3 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package server

import (
	"context"
	"net/http"

	collection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
	"goa.design/plugins/v3/cors"
)

// Server lists the collection service endpoint HTTP handlers.
type Server struct {
	Mounts     []*MountPoint
	List       http.Handler
	Show       http.Handler
	Delete     http.Handler
	Cancel     http.Handler
	Retry      http.Handler
	Workflow   http.Handler
	Download   http.Handler
	Decide     http.Handler
	Bulk       http.Handler
	BulkStatus http.Handler
	CORS       http.Handler
}

// ErrorNamer is an interface implemented by generated error structs that
// exposes the name of the error as defined in the design.
type ErrorNamer interface {
	ErrorName() string
}

// MountPoint holds information about the mounted endpoints.
type MountPoint struct {
	// Method is the name of the service method served by the mounted HTTP handler.
	Method string
	// Verb is the HTTP method used to match requests to the mounted handler.
	Verb string
	// Pattern is the HTTP request path pattern used to match requests to the
	// mounted handler.
	Pattern string
}

// New instantiates HTTP handlers for all the collection service endpoints
// using the provided encoder and decoder. The handlers are mounted on the
// given mux using the HTTP verb and path defined in the design. errhandler is
// called whenever a response fails to be encoded. formatter is used to format
// errors returned by the service methods prior to encoding. Both errhandler
// and formatter are optional and can be nil.
func New(
	e *collection.Endpoints,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) *Server {
	return &Server{
		Mounts: []*MountPoint{
			{"List", "GET", "/collection"},
			{"Show", "GET", "/collection/{id}"},
			{"Delete", "DELETE", "/collection/{id}"},
			{"Cancel", "POST", "/collection/{id}/cancel"},
			{"Retry", "POST", "/collection/{id}/retry"},
			{"Workflow", "GET", "/collection/{id}/workflow"},
			{"Download", "GET", "/collection/{id}/download"},
			{"Decide", "POST", "/collection/{id}/decision"},
			{"Bulk", "POST", "/collection/bulk"},
			{"BulkStatus", "GET", "/collection/bulk"},
			{"CORS", "OPTIONS", "/collection"},
			{"CORS", "OPTIONS", "/collection/{id}"},
			{"CORS", "OPTIONS", "/collection/{id}/cancel"},
			{"CORS", "OPTIONS", "/collection/{id}/retry"},
			{"CORS", "OPTIONS", "/collection/{id}/workflow"},
			{"CORS", "OPTIONS", "/collection/{id}/download"},
			{"CORS", "OPTIONS", "/collection/{id}/decision"},
			{"CORS", "OPTIONS", "/collection/bulk"},
		},
		List:       NewListHandler(e.List, mux, decoder, encoder, errhandler, formatter),
		Show:       NewShowHandler(e.Show, mux, decoder, encoder, errhandler, formatter),
		Delete:     NewDeleteHandler(e.Delete, mux, decoder, encoder, errhandler, formatter),
		Cancel:     NewCancelHandler(e.Cancel, mux, decoder, encoder, errhandler, formatter),
		Retry:      NewRetryHandler(e.Retry, mux, decoder, encoder, errhandler, formatter),
		Workflow:   NewWorkflowHandler(e.Workflow, mux, decoder, encoder, errhandler, formatter),
		Download:   NewDownloadHandler(e.Download, mux, decoder, encoder, errhandler, formatter),
		Decide:     NewDecideHandler(e.Decide, mux, decoder, encoder, errhandler, formatter),
		Bulk:       NewBulkHandler(e.Bulk, mux, decoder, encoder, errhandler, formatter),
		BulkStatus: NewBulkStatusHandler(e.BulkStatus, mux, decoder, encoder, errhandler, formatter),
		CORS:       NewCORSHandler(),
	}
}

// Service returns the name of the service served.
func (s *Server) Service() string { return "collection" }

// Use wraps the server handlers with the given middleware.
func (s *Server) Use(m func(http.Handler) http.Handler) {
	s.List = m(s.List)
	s.Show = m(s.Show)
	s.Delete = m(s.Delete)
	s.Cancel = m(s.Cancel)
	s.Retry = m(s.Retry)
	s.Workflow = m(s.Workflow)
	s.Download = m(s.Download)
	s.Decide = m(s.Decide)
	s.Bulk = m(s.Bulk)
	s.BulkStatus = m(s.BulkStatus)
	s.CORS = m(s.CORS)
}

// Mount configures the mux to serve the collection endpoints.
func Mount(mux goahttp.Muxer, h *Server) {
	MountListHandler(mux, h.List)
	MountShowHandler(mux, h.Show)
	MountDeleteHandler(mux, h.Delete)
	MountCancelHandler(mux, h.Cancel)
	MountRetryHandler(mux, h.Retry)
	MountWorkflowHandler(mux, h.Workflow)
	MountDownloadHandler(mux, h.Download)
	MountDecideHandler(mux, h.Decide)
	MountBulkHandler(mux, h.Bulk)
	MountBulkStatusHandler(mux, h.BulkStatus)
	MountCORSHandler(mux, h.CORS)
}

// MountListHandler configures the mux to serve the "collection" service "list"
// endpoint.
func MountListHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/collection", f)
}

// NewListHandler creates a HTTP handler which loads the HTTP request and calls
// the "collection" service "list" endpoint.
func NewListHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeListRequest(mux, decoder)
		encodeResponse = EncodeListResponse(encoder)
		encodeError    = goahttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "list")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountShowHandler configures the mux to serve the "collection" service "show"
// endpoint.
func MountShowHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/collection/{id}", f)
}

// NewShowHandler creates a HTTP handler which loads the HTTP request and calls
// the "collection" service "show" endpoint.
func NewShowHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeShowRequest(mux, decoder)
		encodeResponse = EncodeShowResponse(encoder)
		encodeError    = EncodeShowError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "show")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountDeleteHandler configures the mux to serve the "collection" service
// "delete" endpoint.
func MountDeleteHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("DELETE", "/collection/{id}", f)
}

// NewDeleteHandler creates a HTTP handler which loads the HTTP request and
// calls the "collection" service "delete" endpoint.
func NewDeleteHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeDeleteRequest(mux, decoder)
		encodeResponse = EncodeDeleteResponse(encoder)
		encodeError    = EncodeDeleteError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "delete")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountCancelHandler configures the mux to serve the "collection" service
// "cancel" endpoint.
func MountCancelHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("POST", "/collection/{id}/cancel", f)
}

// NewCancelHandler creates a HTTP handler which loads the HTTP request and
// calls the "collection" service "cancel" endpoint.
func NewCancelHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeCancelRequest(mux, decoder)
		encodeResponse = EncodeCancelResponse(encoder)
		encodeError    = EncodeCancelError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "cancel")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountRetryHandler configures the mux to serve the "collection" service
// "retry" endpoint.
func MountRetryHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("POST", "/collection/{id}/retry", f)
}

// NewRetryHandler creates a HTTP handler which loads the HTTP request and
// calls the "collection" service "retry" endpoint.
func NewRetryHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeRetryRequest(mux, decoder)
		encodeResponse = EncodeRetryResponse(encoder)
		encodeError    = EncodeRetryError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "retry")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountWorkflowHandler configures the mux to serve the "collection" service
// "workflow" endpoint.
func MountWorkflowHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/collection/{id}/workflow", f)
}

// NewWorkflowHandler creates a HTTP handler which loads the HTTP request and
// calls the "collection" service "workflow" endpoint.
func NewWorkflowHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeWorkflowRequest(mux, decoder)
		encodeResponse = EncodeWorkflowResponse(encoder)
		encodeError    = EncodeWorkflowError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "workflow")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountDownloadHandler configures the mux to serve the "collection" service
// "download" endpoint.
func MountDownloadHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/collection/{id}/download", f)
}

// NewDownloadHandler creates a HTTP handler which loads the HTTP request and
// calls the "collection" service "download" endpoint.
func NewDownloadHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeDownloadRequest(mux, decoder)
		encodeResponse = EncodeDownloadResponse(encoder)
		encodeError    = EncodeDownloadError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "download")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountDecideHandler configures the mux to serve the "collection" service
// "decide" endpoint.
func MountDecideHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("POST", "/collection/{id}/decision", f)
}

// NewDecideHandler creates a HTTP handler which loads the HTTP request and
// calls the "collection" service "decide" endpoint.
func NewDecideHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeDecideRequest(mux, decoder)
		encodeResponse = EncodeDecideResponse(encoder)
		encodeError    = EncodeDecideError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "decide")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountBulkHandler configures the mux to serve the "collection" service "bulk"
// endpoint.
func MountBulkHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("POST", "/collection/bulk", f)
}

// NewBulkHandler creates a HTTP handler which loads the HTTP request and calls
// the "collection" service "bulk" endpoint.
func NewBulkHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		decodeRequest  = DecodeBulkRequest(mux, decoder)
		encodeResponse = EncodeBulkResponse(encoder)
		encodeError    = EncodeBulkError(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "bulk")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		payload, err := decodeRequest(r)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		res, err := endpoint(ctx, payload)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountBulkStatusHandler configures the mux to serve the "collection" service
// "bulk_status" endpoint.
func MountBulkStatusHandler(mux goahttp.Muxer, h http.Handler) {
	f, ok := HandleCollectionOrigin(h).(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("GET", "/collection/bulk", f)
}

// NewBulkStatusHandler creates a HTTP handler which loads the HTTP request and
// calls the "collection" service "bulk_status" endpoint.
func NewBulkStatusHandler(
	endpoint goa.Endpoint,
	mux goahttp.Muxer,
	decoder func(*http.Request) goahttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) goahttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(err error) goahttp.Statuser,
) http.Handler {
	var (
		encodeResponse = EncodeBulkStatusResponse(encoder)
		encodeError    = goahttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), goahttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, goa.MethodKey, "bulk_status")
		ctx = context.WithValue(ctx, goa.ServiceKey, "collection")
		var err error
		res, err := endpoint(ctx, nil)
		if err != nil {
			if err := encodeError(ctx, w, err); err != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		if err := encodeResponse(ctx, w, res); err != nil {
			errhandler(ctx, w, err)
		}
	})
}

// MountCORSHandler configures the mux to serve the CORS endpoints for the
// service collection.
func MountCORSHandler(mux goahttp.Muxer, h http.Handler) {
	h = HandleCollectionOrigin(h)
	f, ok := h.(http.HandlerFunc)
	if !ok {
		f = func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
	mux.Handle("OPTIONS", "/collection", f)
	mux.Handle("OPTIONS", "/collection/{id}", f)
	mux.Handle("OPTIONS", "/collection/{id}/cancel", f)
	mux.Handle("OPTIONS", "/collection/{id}/retry", f)
	mux.Handle("OPTIONS", "/collection/{id}/workflow", f)
	mux.Handle("OPTIONS", "/collection/{id}/download", f)
	mux.Handle("OPTIONS", "/collection/{id}/decision", f)
	mux.Handle("OPTIONS", "/collection/bulk", f)
}

// NewCORSHandler creates a HTTP handler which returns a simple 200 response.
func NewCORSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
}

// HandleCollectionOrigin applies the CORS response headers corresponding to
// the origin for the service collection.
func HandleCollectionOrigin(h http.Handler) http.Handler {
	origHndlr := h.(http.HandlerFunc)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			origHndlr(w, r)
			return
		}
		if cors.MatchOrigin(origin, "*") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			if acrm := r.Header.Get("Access-Control-Request-Method"); acrm != "" {
				// We are handling a preflight request
				w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS")
			}
			origHndlr(w, r)
			return
		}
		origHndlr(w, r)
		return
	})
}
