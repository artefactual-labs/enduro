// Code generated by goa v3.5.1, DO NOT EDIT.
//
// collection HTTP client encoders and decoders
//
// Command:
// $ goa-v3.4.3 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	collection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	collectionviews "github.com/artefactual-labs/enduro/internal/api/gen/collection/views"
	goahttp "goa.design/goa/v3/http"
)

// BuildListRequest instantiates a HTTP request object with method and path set
// to call the "collection" service "list" endpoint
func (c *Client) BuildListRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ListCollectionPath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "list", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// EncodeListRequest returns an encoder for requests sent to the collection
// list server.
func EncodeListRequest(encoder func(*http.Request) goahttp.Encoder) func(*http.Request, interface{}) error {
	return func(req *http.Request, v interface{}) error {
		p, ok := v.(*collection.ListPayload)
		if !ok {
			return goahttp.ErrInvalidType("collection", "list", "*collection.ListPayload", v)
		}
		values := req.URL.Query()
		if p.Name != nil {
			values.Add("name", *p.Name)
		}
		if p.OriginalID != nil {
			values.Add("original_id", *p.OriginalID)
		}
		if p.TransferID != nil {
			values.Add("transfer_id", *p.TransferID)
		}
		if p.AipID != nil {
			values.Add("aip_id", *p.AipID)
		}
		if p.PipelineID != nil {
			values.Add("pipeline_id", *p.PipelineID)
		}
		if p.EarliestCreatedTime != nil {
			values.Add("earliest_created_time", *p.EarliestCreatedTime)
		}
		if p.LatestCreatedTime != nil {
			values.Add("latest_created_time", *p.LatestCreatedTime)
		}
		if p.Status != nil {
			values.Add("status", *p.Status)
		}
		if p.Cursor != nil {
			values.Add("cursor", *p.Cursor)
		}
		req.URL.RawQuery = values.Encode()
		return nil
	}
}

// DecodeListResponse returns a decoder for responses returned by the
// collection list endpoint. restoreBody controls whether the response body
// should be restored after having been read.
func DecodeListResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
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
				return nil, goahttp.ErrDecodingError("collection", "list", err)
			}
			err = ValidateListResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "list", err)
			}
			res := NewListResultOK(&body)
			return res, nil
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "list", resp.StatusCode, string(body))
		}
	}
}

// BuildShowRequest instantiates a HTTP request object with method and path set
// to call the "collection" service "show" endpoint
func (c *Client) BuildShowRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	var (
		id uint
	)
	{
		p, ok := v.(*collection.ShowPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("collection", "show", "*collection.ShowPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ShowCollectionPath(id)}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "show", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeShowResponse returns a decoder for responses returned by the
// collection show endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeShowResponse may return the following errors:
//	- "not_found" (type *collection.NotFound): http.StatusNotFound
//	- error: internal error
func DecodeShowResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
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
				return nil, goahttp.ErrDecodingError("collection", "show", err)
			}
			p := NewShowEnduroStoredCollectionOK(&body)
			view := "default"
			vres := &collectionviews.EnduroStoredCollection{Projected: p, View: view}
			if err = collectionviews.ValidateEnduroStoredCollection(vres); err != nil {
				return nil, goahttp.ErrValidationError("collection", "show", err)
			}
			res := collection.NewEnduroStoredCollection(vres)
			return res, nil
		case http.StatusNotFound:
			var (
				body ShowNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "show", err)
			}
			err = ValidateShowNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "show", err)
			}
			return nil, NewShowNotFound(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "show", resp.StatusCode, string(body))
		}
	}
}

// BuildDeleteRequest instantiates a HTTP request object with method and path
// set to call the "collection" service "delete" endpoint
func (c *Client) BuildDeleteRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	var (
		id uint
	)
	{
		p, ok := v.(*collection.DeletePayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("collection", "delete", "*collection.DeletePayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: DeleteCollectionPath(id)}
	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "delete", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeDeleteResponse returns a decoder for responses returned by the
// collection delete endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeDeleteResponse may return the following errors:
//	- "not_found" (type *collection.NotFound): http.StatusNotFound
//	- error: internal error
func DecodeDeleteResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusNoContent:
			return nil, nil
		case http.StatusNotFound:
			var (
				body DeleteNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "delete", err)
			}
			err = ValidateDeleteNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "delete", err)
			}
			return nil, NewDeleteNotFound(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "delete", resp.StatusCode, string(body))
		}
	}
}

// BuildCancelRequest instantiates a HTTP request object with method and path
// set to call the "collection" service "cancel" endpoint
func (c *Client) BuildCancelRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	var (
		id uint
	)
	{
		p, ok := v.(*collection.CancelPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("collection", "cancel", "*collection.CancelPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: CancelCollectionPath(id)}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "cancel", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeCancelResponse returns a decoder for responses returned by the
// collection cancel endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeCancelResponse may return the following errors:
//	- "not_found" (type *collection.NotFound): http.StatusNotFound
//	- "not_running" (type *goa.ServiceError): http.StatusBadRequest
//	- error: internal error
func DecodeCancelResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			return nil, nil
		case http.StatusNotFound:
			var (
				body CancelNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "cancel", err)
			}
			err = ValidateCancelNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "cancel", err)
			}
			return nil, NewCancelNotFound(&body)
		case http.StatusBadRequest:
			var (
				body CancelNotRunningResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "cancel", err)
			}
			err = ValidateCancelNotRunningResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "cancel", err)
			}
			return nil, NewCancelNotRunning(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "cancel", resp.StatusCode, string(body))
		}
	}
}

// BuildRetryRequest instantiates a HTTP request object with method and path
// set to call the "collection" service "retry" endpoint
func (c *Client) BuildRetryRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	var (
		id uint
	)
	{
		p, ok := v.(*collection.RetryPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("collection", "retry", "*collection.RetryPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: RetryCollectionPath(id)}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "retry", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeRetryResponse returns a decoder for responses returned by the
// collection retry endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeRetryResponse may return the following errors:
//	- "not_found" (type *collection.NotFound): http.StatusNotFound
//	- "not_running" (type *goa.ServiceError): http.StatusBadRequest
//	- error: internal error
func DecodeRetryResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			return nil, nil
		case http.StatusNotFound:
			var (
				body RetryNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "retry", err)
			}
			err = ValidateRetryNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "retry", err)
			}
			return nil, NewRetryNotFound(&body)
		case http.StatusBadRequest:
			var (
				body RetryNotRunningResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "retry", err)
			}
			err = ValidateRetryNotRunningResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "retry", err)
			}
			return nil, NewRetryNotRunning(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "retry", resp.StatusCode, string(body))
		}
	}
}

// BuildWorkflowRequest instantiates a HTTP request object with method and path
// set to call the "collection" service "workflow" endpoint
func (c *Client) BuildWorkflowRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	var (
		id uint
	)
	{
		p, ok := v.(*collection.WorkflowPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("collection", "workflow", "*collection.WorkflowPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: WorkflowCollectionPath(id)}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "workflow", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeWorkflowResponse returns a decoder for responses returned by the
// collection workflow endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeWorkflowResponse may return the following errors:
//	- "not_found" (type *collection.NotFound): http.StatusNotFound
//	- error: internal error
func DecodeWorkflowResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body WorkflowResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "workflow", err)
			}
			p := NewWorkflowEnduroCollectionWorkflowStatusOK(&body)
			view := "default"
			vres := &collectionviews.EnduroCollectionWorkflowStatus{Projected: p, View: view}
			if err = collectionviews.ValidateEnduroCollectionWorkflowStatus(vres); err != nil {
				return nil, goahttp.ErrValidationError("collection", "workflow", err)
			}
			res := collection.NewEnduroCollectionWorkflowStatus(vres)
			return res, nil
		case http.StatusNotFound:
			var (
				body WorkflowNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "workflow", err)
			}
			err = ValidateWorkflowNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "workflow", err)
			}
			return nil, NewWorkflowNotFound(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "workflow", resp.StatusCode, string(body))
		}
	}
}

// BuildDownloadRequest instantiates a HTTP request object with method and path
// set to call the "collection" service "download" endpoint
func (c *Client) BuildDownloadRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	var (
		id uint
	)
	{
		p, ok := v.(*collection.DownloadPayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("collection", "download", "*collection.DownloadPayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: DownloadCollectionPath(id)}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "download", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeDownloadResponse returns a decoder for responses returned by the
// collection download endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeDownloadResponse may return the following errors:
//	- "not_found" (type *collection.NotFound): http.StatusNotFound
//	- error: internal error
func DecodeDownloadResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body []byte
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "download", err)
			}
			return body, nil
		case http.StatusNotFound:
			var (
				body DownloadNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "download", err)
			}
			err = ValidateDownloadNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "download", err)
			}
			return nil, NewDownloadNotFound(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "download", resp.StatusCode, string(body))
		}
	}
}

// BuildDecideRequest instantiates a HTTP request object with method and path
// set to call the "collection" service "decide" endpoint
func (c *Client) BuildDecideRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	var (
		id uint
	)
	{
		p, ok := v.(*collection.DecidePayload)
		if !ok {
			return nil, goahttp.ErrInvalidType("collection", "decide", "*collection.DecidePayload", v)
		}
		id = p.ID
	}
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: DecideCollectionPath(id)}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "decide", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// EncodeDecideRequest returns an encoder for requests sent to the collection
// decide server.
func EncodeDecideRequest(encoder func(*http.Request) goahttp.Encoder) func(*http.Request, interface{}) error {
	return func(req *http.Request, v interface{}) error {
		p, ok := v.(*collection.DecidePayload)
		if !ok {
			return goahttp.ErrInvalidType("collection", "decide", "*collection.DecidePayload", v)
		}
		body := p
		if err := encoder(req).Encode(&body); err != nil {
			return goahttp.ErrEncodingError("collection", "decide", err)
		}
		return nil
	}
}

// DecodeDecideResponse returns a decoder for responses returned by the
// collection decide endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeDecideResponse may return the following errors:
//	- "not_found" (type *collection.NotFound): http.StatusNotFound
//	- "not_valid" (type *goa.ServiceError): http.StatusBadRequest
//	- error: internal error
func DecodeDecideResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			return nil, nil
		case http.StatusNotFound:
			var (
				body DecideNotFoundResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "decide", err)
			}
			err = ValidateDecideNotFoundResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "decide", err)
			}
			return nil, NewDecideNotFound(&body)
		case http.StatusBadRequest:
			var (
				body DecideNotValidResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "decide", err)
			}
			err = ValidateDecideNotValidResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "decide", err)
			}
			return nil, NewDecideNotValid(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "decide", resp.StatusCode, string(body))
		}
	}
}

// BuildBulkRequest instantiates a HTTP request object with method and path set
// to call the "collection" service "bulk" endpoint
func (c *Client) BuildBulkRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: BulkCollectionPath()}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "bulk", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// EncodeBulkRequest returns an encoder for requests sent to the collection
// bulk server.
func EncodeBulkRequest(encoder func(*http.Request) goahttp.Encoder) func(*http.Request, interface{}) error {
	return func(req *http.Request, v interface{}) error {
		p, ok := v.(*collection.BulkPayload)
		if !ok {
			return goahttp.ErrInvalidType("collection", "bulk", "*collection.BulkPayload", v)
		}
		body := NewBulkRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return goahttp.ErrEncodingError("collection", "bulk", err)
		}
		return nil
	}
}

// DecodeBulkResponse returns a decoder for responses returned by the
// collection bulk endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeBulkResponse may return the following errors:
//	- "not_available" (type *goa.ServiceError): http.StatusConflict
//	- "not_valid" (type *goa.ServiceError): http.StatusBadRequest
//	- error: internal error
func DecodeBulkResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusAccepted:
			var (
				body BulkResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "bulk", err)
			}
			err = ValidateBulkResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "bulk", err)
			}
			res := NewBulkResultAccepted(&body)
			return res, nil
		case http.StatusConflict:
			var (
				body BulkNotAvailableResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "bulk", err)
			}
			err = ValidateBulkNotAvailableResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "bulk", err)
			}
			return nil, NewBulkNotAvailable(&body)
		case http.StatusBadRequest:
			var (
				body BulkNotValidResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "bulk", err)
			}
			err = ValidateBulkNotValidResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "bulk", err)
			}
			return nil, NewBulkNotValid(&body)
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "bulk", resp.StatusCode, string(body))
		}
	}
}

// BuildBulkStatusRequest instantiates a HTTP request object with method and
// path set to call the "collection" service "bulk_status" endpoint
func (c *Client) BuildBulkStatusRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: BulkStatusCollectionPath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("collection", "bulk_status", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeBulkStatusResponse returns a decoder for responses returned by the
// collection bulk_status endpoint. restoreBody controls whether the response
// body should be restored after having been read.
func DecodeBulkStatusResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body BulkStatusResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("collection", "bulk_status", err)
			}
			err = ValidateBulkStatusResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("collection", "bulk_status", err)
			}
			res := NewBulkStatusResultOK(&body)
			return res, nil
		default:
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("collection", "bulk_status", resp.StatusCode, string(body))
		}
	}
}

// unmarshalEnduroStoredCollectionResponseBodyToCollectionEnduroStoredCollection
// builds a value of type *collection.EnduroStoredCollection from a value of
// type *EnduroStoredCollectionResponseBody.
func unmarshalEnduroStoredCollectionResponseBodyToCollectionEnduroStoredCollection(v *EnduroStoredCollectionResponseBody) *collection.EnduroStoredCollection {
	res := &collection.EnduroStoredCollection{
		ID:          *v.ID,
		Name:        v.Name,
		Status:      *v.Status,
		WorkflowID:  v.WorkflowID,
		RunID:       v.RunID,
		TransferID:  v.TransferID,
		AipID:       v.AipID,
		OriginalID:  v.OriginalID,
		PipelineID:  v.PipelineID,
		CreatedAt:   *v.CreatedAt,
		StartedAt:   v.StartedAt,
		CompletedAt: v.CompletedAt,
	}

	return res
}

// unmarshalEnduroCollectionWorkflowHistoryResponseBodyToCollectionviewsEnduroCollectionWorkflowHistoryView
// builds a value of type *collectionviews.EnduroCollectionWorkflowHistoryView
// from a value of type *EnduroCollectionWorkflowHistoryResponseBody.
func unmarshalEnduroCollectionWorkflowHistoryResponseBodyToCollectionviewsEnduroCollectionWorkflowHistoryView(v *EnduroCollectionWorkflowHistoryResponseBody) *collectionviews.EnduroCollectionWorkflowHistoryView {
	if v == nil {
		return nil
	}
	res := &collectionviews.EnduroCollectionWorkflowHistoryView{
		ID:      v.ID,
		Type:    v.Type,
		Details: v.Details,
	}

	return res
}
