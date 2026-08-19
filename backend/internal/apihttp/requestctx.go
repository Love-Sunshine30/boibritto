package apihttp

import (
	"context"
	"log/slog"
	"net/http"
)

type ctxKey string

const loggerCtxKey ctxKey = "logger"

// WithLogger attaches a logger to the request context so downstream code
// (like RespondError) can log without needing the logger threaded through
// every function signature.
func WithLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), loggerCtxKey, logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
