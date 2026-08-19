package apihttp

import (
	"errors"
	"net/http"

	"boibritto/internal/apperror"
)

// APIError is the typed error every handler should ultimately respond with.
// Code is a short, stable, machine-readable string (for client-side
// branching); Message is human-readable; Status is the HTTP status to send.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Message }

func ErrNotFound(msg string) *APIError {
	return &APIError{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

func ErrForbidden(msg string) *APIError {
	return &APIError{Status: http.StatusForbidden, Code: "forbidden", Message: msg}
}

func ErrUnauthorized(msg string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: msg}
}

func ErrValidation(msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: "validation_failed", Message: msg}
}

func ErrConflict(msg string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: "conflict", Message: msg}
}

func ErrInternal(msg string) *APIError {
	return &APIError{Status: http.StatusInternalServerError, Code: "internal_error", Message: msg}
}

// toAPIError maps any error into an *APIError. If it's already one, it's
// returned as-is. If it's (or wraps) one of the apperror sentinels that
// service.go files return, it's mapped to the matching APIError. Anything
// else is treated as an unexpected internal error — deliberately vague to
// the client, since arbitrary error strings can leak internal details.
func toAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}

	switch {
	case errors.Is(err, apperror.ErrNotFound):
		return ErrNotFound("resource not found")
	case errors.Is(err, apperror.ErrForbidden):
		return ErrForbidden("you don't have permission to do that")
	case errors.Is(err, apperror.ErrUnauthorized):
		return ErrUnauthorized("authentication required")
	case errors.Is(err, apperror.ErrValidation):
		return ErrValidation("invalid request")
	case errors.Is(err, apperror.ErrConflict):
		return ErrConflict("conflict with existing data")
	default:
		return ErrInternal("something went wrong")
	}
}
