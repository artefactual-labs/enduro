// Code generated by goa v3.11.3, DO NOT EDIT.
//
// pipeline HTTP client encoders and decoders
//
// Command:
// $ goa-v3.11.3 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	pipeline "github.com/artefactual-labs/enduro/internal/api/gen/pipeline"
	pipelineviews "github.com/artefactual-labs/enduro/internal/api/gen/pipeline/views"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// BuildListRequest instantiates a HTTP request object with method and path set
// to call the "pipeline" service "list" endpoint
func (c *Client) BuildListRequest(ctx context.Context, v any) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ListPipelinePath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("pipeline", "list", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// EncodeListRequest returns an encoder for requests sent to the pipeline list
// server.
func EncodeListRequest(encoder func(*http.Request) goahttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*pipeline.ListPayload)
		if !ok {
			return goahttp.ErrInvalidType("pipeline", "list", "*pipeline.ListPayload", v)
		}
		values := req.URL.Query()
		if p.Name != nil {
			values.Add("name", *p.Name)
		}
		values.Add("status", fmt.Sprintf("%v", p.Status))
		req.URL.RawQuery = values.Encode()
		return nil
	}
}

// DecodeListResponse returns a decoder for responses returned by the pipeline
// list endpoint. restoreBody controls whether the response body should be
// restored after having been read.
func DecodeListResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body ListResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("pipeline", "list", err)
			}
			for _, e := range body {
				if e != nil {
					if err2 := ValidateEnduroStoredPipelineResponse(e); err2 != nil {
						err = goa.MergeErrors(err, err2)
					}
				}
			}
			if err != nil {
				return nil, goahttp.ErrValidationError("pipeline", "list", err)
			}
			res := NewListEnduroStoredPipelineOK(body)
			return res, nil
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("pipeline", "list", resp.StatusCode, string(body))
		}
	}
}

// BuildShowRequest instantiates a HTTP request object with method and path set
// to call the "pipeline" service "show" endpoint
func (c *Client) BuildShowRequest(ctx context.Context, v any) (*http.Request, error) {
	var (
		id string
	)
	{
		p, ok := v.(*pipeline.ShowPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("pipeline", "show", "*pipeline.ShowPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ShowPipelinePath(id)}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("pipeline", "show", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeShowResponse returns a decoder for responses returned by the pipeline
// show endpoint. restoreBody controls whether the response body should be
// restored after having been read.
// DecodeShowResponse may return the following errors:
//   - "not_found" (type *pipeline.PipelineNotFound): http.StatusNotFound
//   - error: internal error
func DecodeShowResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body ShowResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("pipeline", "show", err)
			}
			p := NewShowEnduroStoredPipelineOK(&body)
			view := "default"
			vres := &pipelineviews.EnduroStoredPipeline{Projected: p, View: view}
			if err = pipelineviews.ValidateEnduroStoredPipeline(vres); err != nil {
				return nil, goahttp.ErrValidationError("pipeline", "show", err)
			}
			res := pipeline.NewEnduroStoredPipeline(vres)
			return res, nil
		case http.StatusNotFound:
			var (
				body ShowNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("pipeline", "show", err)
			}
			err = ValidateShowNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("pipeline", "show", err)
			}
			return nil, NewShowNotFound(&body)
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("pipeline", "show", resp.StatusCode, string(body))
		}
	}
}

// BuildProcessingRequest instantiates a HTTP request object with method and
// path set to call the "pipeline" service "processing" endpoint
func (c *Client) BuildProcessingRequest(ctx context.Context, v any) (*http.Request, error) {
	var (
		id string
	)
	{
		p, ok := v.(*pipeline.ProcessingPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("pipeline", "processing", "*pipeline.ProcessingPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ProcessingPipelinePath(id)}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("pipeline", "processing", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeProcessingResponse returns a decoder for responses returned by the
// pipeline processing endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeProcessingResponse may return the following errors:
//   - "not_found" (type *pipeline.PipelineNotFound): http.StatusNotFound
//   - error: internal error
func DecodeProcessingResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body []string
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("pipeline", "processing", err)
			}
			return body, nil
		case http.StatusNotFound:
			var (
				body ProcessingNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("pipeline", "processing", err)
			}
			err = ValidateProcessingNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("pipeline", "processing", err)
			}
			return nil, NewProcessingNotFound(&body)
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("pipeline", "processing", resp.StatusCode, string(body))
		}
	}
}

// unmarshalEnduroStoredPipelineResponseToPipelineEnduroStoredPipeline builds a
// value of type *pipeline.EnduroStoredPipeline from a value of type
// *EnduroStoredPipelineResponse.
func unmarshalEnduroStoredPipelineResponseToPipelineEnduroStoredPipeline(v *EnduroStoredPipelineResponse) *pipeline.EnduroStoredPipeline {
	res := &pipeline.EnduroStoredPipeline{
		ID:       v.ID,
		Name:     *v.Name,
		Capacity: v.Capacity,
		Current:  v.Current,
		Status:   v.Status,
	}

	return res
}
