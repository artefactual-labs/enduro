package api

import (
	"net/http"
	"time"
)

func versionHeaderMiddleware(version string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Enduro-Version", version)
			h.ServeHTTP(w, r)
		})
	}
}

// writeTimeout sets the write deadline for writing the response. A zero value
// means no timeout.
func writeTimeout(h http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		var deadline time.Time
		if timeout != 0 {
			deadline = time.Now().Add(timeout)
		}
		_ = rc.SetWriteDeadline(deadline)
		h.ServeHTTP(w, r)
	})
}
