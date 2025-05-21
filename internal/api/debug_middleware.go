// This file is based on `debug.go` from the Goa project:
// https://github.com/goadesign/goa/blob/v3/http/middleware/debug.go
//
// Copyright (c) 2015 Raphaël Simon
// Licensed under the MIT License:
// https://github.com/goadesign/goa/blob/v3/LICENSE
//
// Modifications have been made from the original version. Namely, to solve an
// issue where response bodies of type `application/x-7z-compressed` should not
// be printed, this copy of the middleware was created and modified accordingly,
// as the original debug middleware in the Goa library could not be directly
// altered.

package api

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"

	goahttp "goa.design/goa/v3/http"
	"goa.design/goa/v3/middleware"
)

// debug returns a debug middleware which prints detailed information about
// incoming requests and outgoing responses including all headers, parameters
// and bodies.
func debug(mux goahttp.Muxer, w io.Writer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			buf := &bytes.Buffer{}
			// Request ID
			reqID := r.Context().Value(middleware.RequestIDKey)
			if reqID == nil {
				reqID = shortID()
			}

			// Request URL
			fmt.Fprintf(buf, "> [%s] %s %s", reqID, r.Method, r.URL.String())

			// Request Headers
			keys := make([]string, len(r.Header))
			i := 0
			for k := range r.Header {
				keys[i] = k
				i++
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(buf, "\n> [%s] %s: %s", reqID, k, strings.Join(r.Header[k], ", "))
			}

			// Request parameters
			params := mux.Vars(r)
			keys = make([]string, len(params))
			i = 0
			for k := range params {
				keys[i] = k
				i++
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(buf, "\n> [%s] %s: %s", reqID, k, params[k])
			}

			// Request body
			b, err := io.ReadAll(r.Body)
			if err != nil {
				b = []byte("failed to read body: " + err.Error())
			}
			if len(b) > 0 {
				buf.WriteByte('\n')
				lines := strings.SplitSeq(string(b), "\n")
				for line := range lines {
					fmt.Fprintf(buf, "[%s] %s\n", reqID, line)
				}
			}
			r.Body = io.NopCloser(bytes.NewBuffer(b))

			dupper := &responseDupper{ResponseWriter: rw, Buffer: &bytes.Buffer{}}
			h.ServeHTTP(dupper, r)

			fmt.Fprintf(buf, "\n< [%s] %s", reqID, http.StatusText(dupper.Status))
			keys = make([]string, len(dupper.Header()))
			printResponseBody := true
			i = 0
			for k, v := range dupper.Header() {
				if k == "Content-Type" && len(v) > 0 && v[0] == "application/x-7z-compressed" {
					printResponseBody = false
				}
				keys[i] = k
				i++
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(buf, "\n< [%s] %s: %s", reqID, k, strings.Join(dupper.Header()[k], ", "))
			}
			if printResponseBody {
				buf.WriteByte('\n')
				lines := strings.SplitSeq(dupper.Buffer.String(), "\n")
				for line := range lines {
					fmt.Fprintf(buf, "[%s] %s\n", reqID, line)
				}
			}
			buf.WriteByte('\n')
			_, err = w.Write(buf.Bytes()) // nolint: errcheck
			if err != nil {
				panic(err)
			}
		})
	}
}

// responseDupper tees the response to a buffer and a response writer.
type responseDupper struct {
	http.ResponseWriter
	Buffer *bytes.Buffer
	Status int
}

// Write writes the data to the buffer and connection as part of an HTTP reply.
func (r *responseDupper) Write(b []byte) (int, error) {
	return io.MultiWriter(r.ResponseWriter, r.Buffer).Write(b)
}

// WriteHeader records the status and sends an HTTP response header with status code.
func (r *responseDupper) WriteHeader(s int) {
	r.Status = s
	r.ResponseWriter.WriteHeader(s)
}

// Hijack supports the http.Hijacker interface.
func (r *responseDupper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("debug middleware: inner ResponseWriter cannot be hijacked: %T", r.ResponseWriter)
}

// shortID produces a " unique" 6 bytes long string.
// Do not use as a reliable way to get unique IDs, instead use for things like logging.
func shortID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b) // nolint: errcheck
	return base64.RawURLEncoding.EncodeToString(b)
}
