// internal/apihttp/ratelimit.go
package apihttp

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimit applies a per-IP request limit. Chosen deliberately per-IP,
// not per-user: unauthenticated endpoints (register-adjacent Firebase
// calls aren't ours, but /healthz and any future public routes are) need
// protection before we even know who the caller is. Authenticated routes
// get this AND could later add a per-user limiter on top if IP-sharing
// (NAT, campus wifi) makes per-IP too coarse.
func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return httprate.LimitByIP(requestsPerMinute, time.Minute)
}

// StrictRateLimit is for the most abuse-sensitive routes — write
// operations that are cheap to spam and expensive to clean up after
// (creating books, posting messages, forum posts).
func StrictRateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return httprate.LimitByIP(requestsPerMinute, time.Minute)
}
