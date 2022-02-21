package collection

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-logr/logr"
	goahttp "goa.design/goa/v3/http"

	"github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/artefactual-labs/enduro/internal/api/gen/http/collection/server"
	"github.com/artefactual-labs/enduro/internal/pipeline"
)

const userAgent = "Enduro (ssclient)"

type downloadReverseProxy struct {
	logger logr.Logger
	proxy  *httputil.ReverseProxy
}

func newDownloadReverseProxy(logger logr.Logger) *downloadReverseProxy {
	return &downloadReverseProxy{
		logger: logger,
	}
}

func (dp downloadReverseProxy) build(p *pipeline.Pipeline, pID string) (*httputil.ReverseProxy, error) {
	bu, auth, err := p.SSAccess()
	if err != nil {
		return nil, fmt.Errorf("error loading Storage Service access details: %v", err)
	}

	path := fmt.Sprintf("api/v2/file/%s/download/", pID)
	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("error building URL: %v", err)
	}

	// Rewrite request URL and headers.
	director := func(req *http.Request) {
		req.URL = bu.ResolveReference(rel)
		req.Header.Add("User-Agent", userAgent)
		req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", auth))
	}

	return &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: dp.modifyResponse,
		Transport:      dp.transport(),
		ErrorHandler:   dp.errorHandler,
	}, nil
}

func (dp downloadReverseProxy) transport() http.RoundTripper {
	return http.DefaultTransport
}

func (dp *downloadReverseProxy) modifyResponse(r *http.Response) error {
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("error communicating with Storage Service: %s (%d)", r.Status, r.StatusCode)
	}

	r.Header.Del("Server")

	return nil
}

func (dp *downloadReverseProxy) errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	rw.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(rw, `{"message": "The operation failed unexpectedly. Contact the administrator for more details."}`)
	dp.logger.Info("Download from Storage Service failed", "msg", err.Error())
}

func (p *downloadReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.proxy.ServeHTTP(rw, req)
}

func (svc *collectionImpl) HTTPDownload(mux goahttp.Muxer, dec func(r *http.Request) goahttp.Decoder) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		payload, err := server.DecodeDownloadRequest(mux, dec)(req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		// Look up collection.
		p := payload.(*collection.DownloadPayload)
		col, err := svc.Goa().Show(context.Background(), &collection.ShowPayload{ID: p.ID})
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		if col.PipelineID == nil || *col.PipelineID == "" {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		if col.AipID == nil || *col.AipID == "" {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		pipeline, err := svc.registry.ByID(*col.PipelineID)
		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		proxy, err := svc.downloadProxy.build(pipeline, *col.AipID)
		if err != nil {
			svc.logger.Info("Error buiding download proxy", "msg", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		proxy.ServeHTTP(rw, req)
	}
}
