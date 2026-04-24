package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func securityHeadersMiddleware(csp string) func(http.Handler) http.Handler {
	if csp == "" {
		return func(h http.Handler) http.Handler {
			return h
		}
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", csp)
			h.ServeHTTP(w, r)
		})
	}
}

func crossOriginProtectionMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	cop, enabled, err := newCrossOriginProtection(allowedOrigins)
	if err != nil {
		panic(fmt.Sprintf("invalid API allowed origin: %v", err))
	}
	if !enabled {
		return func(h http.Handler) http.Handler {
			return h
		}
	}

	return cop.Handler
}

func corsResponseHeaderMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := newAllowedOriginSet(allowedOrigins)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || origins.isAllowed(origin, requestOrigin(r)) {
				h.ServeHTTP(w, r)
				return
			}

			h.ServeHTTP(corsHeaderFilteringResponseWriter{ResponseWriter: w}, r)
		})
	}
}

func newCrossOriginProtection(allowedOrigins []string) (*http.CrossOriginProtection, bool, error) {
	cop := http.NewCrossOriginProtection()
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			return nil, false, nil
		}
		if err := cop.AddTrustedOrigin(origin); err != nil {
			return nil, false, fmt.Errorf("%q: %w", origin, err)
		}
	}

	return cop, true, nil
}

type allowedOriginSet struct {
	allowAny bool
	origins  map[string]struct{}
}

func newAllowedOriginSet(allowedOrigins []string) allowedOriginSet {
	origins := allowedOriginSet{origins: make(map[string]struct{}, len(allowedOrigins))}
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			origins.allowAny = true
			continue
		}
		origins.origins[origin] = struct{}{}
	}

	return origins
}

func (o allowedOriginSet) isAllowed(origin, sameOrigin string) bool {
	if o.allowAny {
		return true
	}
	if origin == sameOrigin {
		return true
	}
	_, ok := o.origins[origin]

	return ok
}

type corsHeaderFilteringResponseWriter struct {
	http.ResponseWriter
}

func (w corsHeaderFilteringResponseWriter) WriteHeader(statusCode int) {
	removeCORSResponseHeaders(w.Header())
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w corsHeaderFilteringResponseWriter) Write(b []byte) (int, error) {
	removeCORSResponseHeaders(w.Header())
	return w.ResponseWriter.Write(b)
}

func (w corsHeaderFilteringResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func removeCORSResponseHeaders(h http.Header) {
	h.Del("Access-Control-Allow-Credentials")
	h.Del("Access-Control-Allow-Headers")
	h.Del("Access-Control-Allow-Methods")
	h.Del("Access-Control-Allow-Origin")
	h.Del("Access-Control-Expose-Headers")
	h.Del("Access-Control-Max-Age")
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	host := r.Host
	if host == "" {
		return ""
	}

	return (&url.URL{Scheme: scheme, Host: host}).String()
}
