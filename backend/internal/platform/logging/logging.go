// internal/platform/logging/logging.go
package logging

import (
	"log/slog"

	"github.com/go-chi/httplog/v2"
)

func New(env string) *httplog.Logger {
	opts := httplog.Options{
		LogLevel:        slog.LevelInfo,
		Concise:         env == "prod", // compact JSON in prod, readable text in dev
		JSON:            env == "prod",
		RequestHeaders:  env != "prod",
		ResponseHeaders: false,
		Tags:            map[string]string{"service": "boibritto-api"},
	}
	if env == "dev" {
		opts.LogLevel = slog.LevelDebug
	}
	return httplog.NewLogger("boibritto", opts)
}
