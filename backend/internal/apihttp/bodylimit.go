// internal/apihttp/bodylimit.go
package apihttp

import "net/http"

// MaxBodySize rejects any request body larger than maxBytes, BEFORE a
// handler's json.Decode ever reads it into memory. This must run early in
// the middleware chain — before RateLimit isn't required, but it should
// be applied globally, not per-route, since every POST/PATCH handler in
// every domain currently decodes JSON with no size awareness of its own.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
