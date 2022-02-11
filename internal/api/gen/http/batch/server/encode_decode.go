// Code generated by goa v3.5.5, DO NOT EDIT.
//
// batch HTTP server encoders and decoders
//
// Command:
// $ goa-v3.5.5 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package server

import (
	"context"
	"io"
	"net/http"

	batch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// EncodeSubmitResponse returns an encoder for responses returned by the batch
// submit endpoint.
func EncodeSubmitResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res, _ := v.(*batch.BatchResult)
		enc := encoder(ctx, w)
		body := NewSubmitResponseBody(res)
		w.WriteHeader(http.StatusAccepted)
		return enc.Encode(body)
	}
}

// DecodeSubmitRequest returns a decoder for requests sent to the batch submit
// endpoint.
func DecodeSubmitRequest(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) (interface{}, error) {
	return func(r *http.Request) (interface{}, error) {
		var (
			body SubmitRequestBody
			err  error
		)
		err = decoder(r).Decode(&body)
		if err != nil {
			if err == io.EOF {
				return nil, goa.MissingPayloadError()
			}
			return nil, goa.DecodePayloadError(err.Error())
		}
		err = ValidateSubmitRequestBody(&body)
		if err != nil {
			return nil, err
		}
		payload := NewSubmitPayload(&body)

		return payload, nil
	}
}

// EncodeSubmitError returns an encoder for errors returned by the submit batch
// endpoint.
func EncodeSubmitError(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder, formatter func(err error) goahttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
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
				body = NewSubmitNotAvailableResponseBody(res)
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
				body = NewSubmitNotValidResponseBody(res)
			}
			w.Header().Set("goa-error", res.ErrorName())
			w.WriteHeader(http.StatusBadRequest)
			return enc.Encode(body)
		default:
			return encodeError(ctx, w, v)
		}
	}
}

// EncodeStatusResponse returns an encoder for responses returned by the batch
// status endpoint.
func EncodeStatusResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res, _ := v.(*batch.BatchStatusResult)
		enc := encoder(ctx, w)
		body := NewStatusResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}

// EncodeHintsResponse returns an encoder for responses returned by the batch
// hints endpoint.
func EncodeHintsResponse(encoder func(context.Context, http.ResponseWriter) goahttp.Encoder) func(context.Context, http.ResponseWriter, interface{}) error {
	return func(ctx context.Context, w http.ResponseWriter, v interface{}) error {
		res, _ := v.(*batch.BatchHintsResult)
		enc := encoder(ctx, w)
		body := NewHintsResponseBody(res)
		w.WriteHeader(http.StatusOK)
		return enc.Encode(body)
	}
}
