package apihttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// envelope is the shape every JSON response takes. On success, Data is
// populated and Error is omitted; on failure, the reverse.
type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

// RespondJSON writes a successful response with the given status and payload.
func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		// Nothing more we can do — headers are already sent at this point.
		// A logger isn't threaded through here by design (see note below);
		// this failure mode (broken pipe, client disconnected) is not
		// actionable server-side.
		_ = err
	}
}

// RespondError writes an error response, mapping the given error to the
// appropriate HTTP status and a stable error code via toAPIError.
func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := toAPIError(err)

	if apiErr.Status >= http.StatusInternalServerError {
		if logger, ok := r.Context().Value(loggerCtxKey).(*slog.Logger); ok {
			logger.Error("request failed", "error", err, "path", r.URL.Path, "status", apiErr.Status)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	_ = json.NewEncoder(w).Encode(envelope{Error: apiErr})
}
