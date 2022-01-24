// Code generated by goa v3.5.4, DO NOT EDIT.
//
// collection HTTP server encoders and decoders
//
// Command:
// $ goa-v3.5.4 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package server

import (
	"context"
	"io"
	"net/http"
	"strconv"

	collection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	collectionviews "github.com/artefactual-labs/enduro/internal/api/gen/collection/views"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// EncodeListResponse returns an encoder for responses returned by the
// collection list endpoint.
func EncodeListResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res, _ := v.(*collection.ListResult)
		enc := encoder(ctx, w)
		body := NewListResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeListRequest returns a decoder for requests sent to the collection list
// endpoint.
func DecodeListRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			name                *string
			originalID          *string
			transferID          *string
			aipID               *string
			pipelineID          *string
			earliestCreatedTime *string
			latestCreatedTime   *string
			status              *string
			cursor              *string
			err                 error
		)
		nameRaw := r.URL.Query().Get("name")
		if nameRaw != "" {
			name = &nameRaw
		}
		originalIDRaw := r.URL.Query().Get("original_id")
		if originalIDRaw != "" {
			originalID = &originalIDRaw
		}
		transferIDRaw := r.URL.Query().Get("transfer_id")
		if transferIDRaw != "" {
			transferID = &transferIDRaw
		}
		if transferID != nil {
			err = goa.MergeErrors(err, goa.ValidateFormat("transferID", *transferID, goa.FormatUUID))
		}
		aipIDRaw := r.URL.Query().Get("aip_id")
		if aipIDRaw != "" {
			aipID = &aipIDRaw
		}
		if aipID != nil {
			err = goa.MergeErrors(err, goa.ValidateFormat("aipID", *aipID, goa.FormatUUID))
		}
		pipelineIDRaw := r.URL.Query().Get("pipeline_id")
		if pipelineIDRaw != "" {
			pipelineID = &pipelineIDRaw
		}
		if pipelineID != nil {
			err = goa.MergeErrors(err, goa.ValidateFormat("pipelineID", *pipelineID, goa.FormatUUID))
		}
		earliestCreatedTimeRaw := r.URL.Query().Get("earliest_created_time")
		if earliestCreatedTimeRaw != "" {
			earliestCreatedTime = &earliestCreatedTimeRaw
		}
		if earliestCreatedTime != nil {
			err = goa.MergeErrors(err, goa.ValidateFormat("earliestCreatedTime", *earliestCreatedTime, goa.FormatDateTime))
		}
		latestCreatedTimeRaw := r.URL.Query().Get("latest_created_time")
		if latestCreatedTimeRaw != "" {
			latestCreatedTime = &latestCreatedTimeRaw
		}
		if latestCreatedTime != nil {
			err = goa.MergeErrors(err, goa.ValidateFormat("latestCreatedTime", *latestCreatedTime, goa.FormatDateTime))
		}
		statusRaw := r.URL.Query().Get("status")
		if statusRaw != "" {
			status = &statusRaw
		}
		if status != nil {
			if !(*status == "new" || *status == "in progress" || *status == "done" || *status == "error" || *status == "unknown" || *status == "queued" || *status == "pending" || *status == "abandoned") {
				err = goa.MergeErrors(err, goa.InvalidEnumValueError("status", *status, []interface{}{"new", "in progress", "done", "error", "unknown", "queued", "pending", "abandoned"}))
			}
		}
		cursorRaw := r.URL.Query().Get("cursor")
		if cursorRaw != "" {
			cursor = &cursorRaw
		}
		if err != nil {
			return nil, err
		}
		payload := NewListPayload(name, originalID, transferID, aipID, pipelineID, earliestCreatedTime, latestCreatedTime, status, cursor)

		return payload, nil
	}
}

// EncodeShowResponse returns an encoder for responses returned by the
// collection show endpoint.
func EncodeShowResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res := v.(*collectionviews.EnduroStoredCollection)
		enc := encoder(ctx, w)
		body := NewShowResponseBody(res.Projected)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeShowRequest returns a decoder for requests sent to the collection show
// endpoint.
func DecodeShowRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			id  uint
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseUint(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "unsigned integer"))
			}
			id = uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewShowPayload(id)

		return payload, nil
	}
}

// EncodeShowError returns an encoder for errors returned by the show
// collection endpoint.
func EncodeShowError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_found":
			res := v.(*collection.CollectionNotfound)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewShowNotFoundResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusNotFound)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeDeleteResponse returns an encoder for responses returned by the
// collection delete endpoint.
func EncodeDeleteResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

// DecodeDeleteRequest returns a decoder for requests sent to the collection
// delete endpoint.
func DecodeDeleteRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			id  uint
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseUint(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "unsigned integer"))
			}
			id = uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewDeletePayload(id)

		return payload, nil
	}
}

// EncodeDeleteError returns an encoder for errors returned by the delete
// collection endpoint.
func EncodeDeleteError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_found":
			res := v.(*collection.CollectionNotfound)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewDeleteNotFoundResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusNotFound)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeCancelResponse returns an encoder for responses returned by the
// collection cancel endpoint.
func EncodeCancelResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		w.WriteHeader(http.StatusOK)
		return nil
	}
}

// DecodeCancelRequest returns a decoder for requests sent to the collection
// cancel endpoint.
func DecodeCancelRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			id  uint
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseUint(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "unsigned integer"))
			}
			id = uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewCancelPayload(id)

		return payload, nil
	}
}

// EncodeCancelError returns an encoder for errors returned by the cancel
// collection endpoint.
func EncodeCancelError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_found":
			res := v.(*collection.CollectionNotfound)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewCancelNotFoundResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusNotFound)
			return enc.Encode(body)
		case "not_running":
			res := v.(*goa.ServiceError)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewCancelNotRunningResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusBadRequest)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeRetryResponse returns an encoder for responses returned by the
// collection retry endpoint.
func EncodeRetryResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		w.WriteHeader(http.StatusOK)
		return nil
	}
}

// DecodeRetryRequest returns a decoder for requests sent to the collection
// retry endpoint.
func DecodeRetryRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			id  uint
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseUint(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "unsigned integer"))
			}
			id = uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewRetryPayload(id)

		return payload, nil
	}
}

// EncodeRetryError returns an encoder for errors returned by the retry
// collection endpoint.
func EncodeRetryError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_found":
			res := v.(*collection.CollectionNotfound)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewRetryNotFoundResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusNotFound)
			return enc.Encode(body)
		case "not_running":
			res := v.(*goa.ServiceError)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewRetryNotRunningResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusBadRequest)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeWorkflowResponse returns an encoder for responses returned by the
// collection workflow endpoint.
func EncodeWorkflowResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res := v.(*collectionviews.EnduroCollectionWorkflowStatus)
		enc := encoder(ctx, w)
		body := NewWorkflowResponseBody(res.Projected)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeWorkflowRequest returns a decoder for requests sent to the collection
// workflow endpoint.
func DecodeWorkflowRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			id  uint
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseUint(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "unsigned integer"))
			}
			id = uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewWorkflowPayload(id)

		return payload, nil
	}
}

// EncodeWorkflowError returns an encoder for errors returned by the workflow
// collection endpoint.
func EncodeWorkflowError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_found":
			res := v.(*collection.CollectionNotfound)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewWorkflowNotFoundResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusNotFound)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeDownloadResponse returns an encoder for responses returned by the
// collection download endpoint.
func EncodeDownloadResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res, _ := v.([]byte)
		enc := encoder(ctx, w)
		body := res
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// DecodeDownloadRequest returns a decoder for requests sent to the collection
// download endpoint.
func DecodeDownloadRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			id  uint
			err error

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseUint(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "unsigned integer"))
			}
			id = uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewDownloadPayload(id)

		return payload, nil
	}
}

// EncodeDownloadError returns an encoder for errors returned by the download
// collection endpoint.
func EncodeDownloadError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_found":
			res := v.(*collection.CollectionNotfound)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewDownloadNotFoundResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusNotFound)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeDecideResponse returns an encoder for responses returned by the
// collection decide endpoint.
func EncodeDecideResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		w.WriteHeader(http.StatusOK)
		return nil
	}
}

// DecodeDecideRequest returns a decoder for requests sent to the collection
// decide endpoint.
func DecodeDecideRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			body struct {
				// Decision option to proceed with
				Option *string `form:"option" json:"option" xml:"option"`
			}
			err error
		)
		err = decoder(r).Decode(&body)
		if err != nil {
			if err == io.EOF {
				return nil, goa.MissingPayloadError()
			}
			return nil, goa.DecodePayloadError(err.Error())
		}

		var (
			id uint

			params = mux.Vars(r)
		)
		{
			idRaw := params["id"]
			v, err2 := strconv.ParseUint(idRaw, 10, strconv.IntSize)
			if err2 != nil {
				err = goa.MergeErrors(err, goa.InvalidFieldTypeError("id", idRaw, "unsigned integer"))
			}
			id = uint(v)
		}
		if err != nil {
			return nil, err
		}
		payload := NewDecidePayload(body, id)

		return payload, nil
	}
}

// EncodeDecideError returns an encoder for errors returned by the decide
// collection endpoint.
func EncodeDecideError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_found":
			res := v.(*collection.CollectionNotfound)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewDecideNotFoundResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusNotFound)
			return enc.Encode(body)
		case "not_valid":
			res := v.(*goa.ServiceError)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewDecideNotValidResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusBadRequest)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeBulkResponse returns an encoder for responses returned by the
// collection bulk endpoint.
func EncodeBulkResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res, _ := v.(*collection.BulkResult)
		enc := encoder(ctx, w)
		body := NewBulkResponseBody(res)
		w.WriteHeader(http.StatusAccepted)
		return enc.Encode(body)
	}
}

// DecodeBulkRequest returns a decoder for requests sent to the collection bulk
// endpoint.
func DecodeBulkRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			body BulkRequestBody
			err  error
		)
		err = decoder(r).Decode(&body)
		if err != nil {
			if err == io.EOF {
				return nil, goa.MissingPayloadError()
			}
			return nil, goa.DecodePayloadError(err.Error())
		}
		err = ValidateBulkRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewBulkPayload(&body)

		return payload, nil
	}
}

// EncodeBulkError returns an encoder for errors returned by the bulk
// collection endpoint.
func EncodeBulkError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := goahttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		en, ok := v.(ErrorNamer)
		if !ok {
			return encodeError(ctx, w, v)
		}
		switch en.ErrorName() {
		case "not_available":
			res := v.(*goa.ServiceError)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewBulkNotAvailableResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusConflict)
			return enc.Encode(body)
		case "not_valid":
			res := v.(*goa.ServiceError)
			enc := encoder(ctx, w)
			var body interface{}
			if formatter != nil {
				body = formatter(res)
			} else {
				body = NewBulkNotValidResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusBadRequest)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeBulkStatusResponse returns an encoder for responses returned by the
// collection bulk_status endpoint.
func EncodeBulkStatusResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res, _ := v.(*collection.BulkStatusResult)
		enc := encoder(ctx, w)
		body := NewBulkStatusResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// marshalCollectionviewsEnduroStoredCollectionViewToEnduroStoredCollectionResponseBody
// builds a value of type *EnduroStoredCollectionResponseBody from a value of
// type *collectionviews.EnduroStoredCollectionView.
func marshalCollectionviewsEnduroStoredCollectionViewToEnduroStoredCollectionResponseBody(v *collectionviews.EnduroStoredCollectionView) *EnduroStoredCollectionResponseBody {
	if v == nil {
		return nil
	}
	res := &EnduroStoredCollectionResponseBody{
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

// marshalCollectionEnduroStoredCollectionToEnduroStoredCollectionResponseBody
// builds a value of type *EnduroStoredCollectionResponseBody from a value of
// type *collection.EnduroStoredCollection.
func marshalCollectionEnduroStoredCollectionToEnduroStoredCollectionResponseBody(v *collection.EnduroStoredCollection) *EnduroStoredCollectionResponseBody {
	res := &EnduroStoredCollectionResponseBody{
		ID:          v.ID,
		Name:        v.Name,
		Status:      v.Status,
		WorkflowID:  v.WorkflowID,
		RunID:       v.RunID,
		TransferID:  v.TransferID,
		AipID:       v.AipID,
		OriginalID:  v.OriginalID,
		PipelineID:  v.PipelineID,
		CreatedAt:   v.CreatedAt,
		StartedAt:   v.StartedAt,
		CompletedAt: v.CompletedAt,
	}

	return res
}

// marshalCollectionviewsEnduroCollectionWorkflowHistoryViewToEnduroCollectionWorkflowHistoryResponseBody
// builds a value of type *EnduroCollectionWorkflowHistoryResponseBody from a
// value of type *collectionviews.EnduroCollectionWorkflowHistoryView.
func marshalCollectionviewsEnduroCollectionWorkflowHistoryViewToEnduroCollectionWorkflowHistoryResponseBody(v *collectionviews.EnduroCollectionWorkflowHistoryView) *EnduroCollectionWorkflowHistoryResponseBody {
	if v == nil {
		return nil
	}
	res := &EnduroCollectionWorkflowHistoryResponseBody{
		ID:      v.ID,
		Type:    v.Type,
		Details: v.Details,
	}

	return res
}
