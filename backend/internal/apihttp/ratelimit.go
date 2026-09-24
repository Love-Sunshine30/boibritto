// internal/apihttp/ratelimit.go
package apihttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func TrustedProxyIP(trustedProxyCIDRs ...string) func(http.Handler) http.Handler {
	return middleware.ClientIPFromXFF(trustedProxyCIDRs...)
}

func clientIPKey(r *http.Request) (string, error) {
	ip := middleware.GetClientIP(r.Context())
	return httprate.CanonicalizeIP(ip), nil
}

func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return httprate.LimitBy(
		requestsPerMinute, time.Minute,
		clientIPKey,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, r, ErrTooManyRequests("rate limit exceeded, please slow down"))
		}),
	)
}

func StrictRateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return httprate.LimitBy(
		requestsPerMinute, time.Minute,
		clientIPKey,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, r, ErrTooManyRequests("rate limit exceeded, please slow down"))
		}),
	)
}
