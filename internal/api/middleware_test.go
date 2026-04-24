package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		csp         string
		wantHeaders map[string]string
	}{
		{
			name: "Adds headers when CSP is configured",
			csp:  "default-src 'self'",
			wantHeaders: map[string]string{
				"Content-Security-Policy": "default-src 'self'",
				"X-Content-Type-Options":  "nosniff",
				"Referrer-Policy":         "same-origin",
				"X-Frame-Options":         "DENY",
			},
		},
		{
			name: "Preserves previous header behavior when CSP is empty",
			wantHeaders: map[string]string{
				"Content-Security-Policy": "",
				"X-Content-Type-Options":  "",
				"Referrer-Policy":         "",
				"X-Frame-Options":         "",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := securityHeadersMiddleware(tc.csp)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			assert.Equal(t, rec.Code, http.StatusNoContent)
			assertHeaders(t, rec.Header(), tc.wantHeaders)
		})
	}
}

func TestCrossOriginProtectionMiddleware(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	cases := []struct {
		name           string
		method         string
		url            string
		allowedOrigins []string
		reqHeaders     map[string]string
		wantStatus     int
	}{
		{
			name:       "Allows unsafe requests without browser origin signals",
			method:     http.MethodPost,
			url:        "http://example.com/collection",
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "Allows same-origin requests",
			method: http.MethodPost,
			url:    "http://example.com/collection",
			reqHeaders: map[string]string{
				"Origin": "http://example.com",
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:           "Allows configured origins",
			method:         http.MethodPost,
			url:            "http://api.example.org/collection",
			allowedOrigins: []string{"https://dashboard.example.org"},
			reqHeaders: map[string]string{
				"Origin": "https://dashboard.example.org",
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:           "Allows any origin when explicitly configured",
			method:         http.MethodPost,
			url:            "http://api.example.org/collection",
			allowedOrigins: []string{"*"},
			reqHeaders: map[string]string{
				"Origin": "https://untrusted.example.org",
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "Rejects unsafe requests from disallowed origins",
			method: http.MethodPost,
			url:    "http://api.example.org/collection",
			reqHeaders: map[string]string{
				"Origin": "https://untrusted.example.org",
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "Allows safe requests from disallowed origins",
			method: http.MethodGet,
			url:    "http://api.example.org/collection",
			reqHeaders: map[string]string{
				"Origin": "https://untrusted.example.org",
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "Rejects unsafe cross-site fetch metadata",
			method: http.MethodPost,
			url:    "http://api.example.org/collection",
			reqHeaders: map[string]string{
				"Sec-Fetch-Site": "cross-site",
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := crossOriginProtectionMiddleware(tc.allowedOrigins)(next)
			req := httptest.NewRequest(tc.method, tc.url, nil)
			setHeaders(req.Header, tc.reqHeaders)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			assert.Equal(t, rec.Code, tc.wantStatus)
		})
	}
}

func TestCORSResponseHeaderMiddleware(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Expose-Headers", "X-Enduro-Version")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.WriteHeader(http.StatusNoContent)
	})

	cases := []struct {
		name           string
		allowedOrigins []string
		origin         string
		wantHeaders    map[string]string
	}{
		{
			name:           "Keeps CORS headers for configured origins",
			allowedOrigins: []string{"https://dashboard.example.org"},
			origin:         "https://dashboard.example.org",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "https://dashboard.example.org",
				"Access-Control-Allow-Methods": "GET, POST",
			},
		},
		{
			name:   "Keeps CORS headers for same-origin requests",
			origin: "http://api.example.org",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "http://api.example.org",
			},
		},
		{
			name:   "Removes CORS headers for disallowed origins",
			origin: "https://untrusted.example.org",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Credentials": "",
				"Access-Control-Allow-Headers":     "",
				"Access-Control-Allow-Methods":     "",
				"Access-Control-Allow-Origin":      "",
				"Access-Control-Expose-Headers":    "",
				"Access-Control-Max-Age":           "",
			},
		},
		{
			name:           "Keeps CORS headers when explicitly disabled",
			allowedOrigins: []string{"*"},
			origin:         "https://untrusted.example.org",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://untrusted.example.org",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := corsResponseHeaderMiddleware(tc.allowedOrigins)(next)
			req := httptest.NewRequest(http.MethodGet, "http://api.example.org/collection", nil)
			req.Header.Set("Origin", tc.origin)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			assert.Equal(t, rec.Code, http.StatusNoContent)
			assertHeaders(t, rec.Header(), tc.wantHeaders)
		})
	}
}

func setHeaders(h http.Header, headers map[string]string) {
	for k, v := range headers {
		h.Set(k, v)
	}
}

func assertHeaders(t *testing.T, h http.Header, headers map[string]string) {
	t.Helper()

	for k, want := range headers {
		assert.Equal(t, h.Get(k), want)
	}
}
