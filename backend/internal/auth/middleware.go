package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	fbauth "firebase.google.com/go/v4/auth"

	"boibritto/internal/apihttp"
	"boibritto/internal/domain"
)

type ctxKey string

const userCtxKey ctxKey = "auth_user"

// RequireAuth verifies the Firebase ID token on every request's
// Authorization header, JIT-provisions a local user record on first sign-in,
// and injects the resulting domain.User into the request context.
func RequireAuth(fbAuth *fbauth.Client, store *Store, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractBearerToken(r)
			if err != nil {
				apihttp.RespondError(w, r, apihttp.ErrUnauthorized("missing or malformed authorization header"))
				return
			}

			verified, err := fbAuth.VerifyIDToken(r.Context(), token)
			if err != nil {
				logger.Warn("token verification failed", "error", err)
				apihttp.RespondError(w, r, apihttp.ErrUnauthorized("invalid or expired token"))
				return
			}

			email, _ := verified.Claims["email"].(string)
			name, _ := verified.Claims["name"].(string)

			user, err := store.FindOrCreateUser(r.Context(), verified.UID, name, email)
			if err != nil {
				logger.Error("failed to provision user", "error", err, "firebase_uid", verified.UID)
				apihttp.RespondError(w, r, apihttp.ErrInternal("could not resolve user"))
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin builds on RequireAuth's injected user — mount it AFTER
// RequireAuth in the middleware chain. Checks a real flag on our own user
// record, not just "is anyone logged in."
func RequireAdmin(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || !user.IsAdmin {
				apihttp.RespondError(w, r, apihttp.ErrForbidden("admin access required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserFromContext retrieves the authenticated user injected by RequireAuth.
// Handlers use this instead of re-parsing tokens or touching the store directly.
func UserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userCtxKey).(*domain.User)
	return user, ok
}

func extractBearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errMissingToken
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", errMissingToken
	}
	return token, nil
}

var errMissingToken = &missingTokenError{}

type missingTokenError struct{}

func (e *missingTokenError) Error() string { return "missing bearer token" }
