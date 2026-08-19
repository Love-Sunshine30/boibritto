package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"

	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
	"boibritto/internal/books"
)

func NewRouter(app *app.App) chi.Router {
	r := chi.NewRouter()

	// --- Global, unauthenticated middleware ---
	r.Use(middleware.RequestID)
	r.Use(httplog.RequestLogger(app.Logger))
	r.Use(middleware.Recoverer)
	r.Use(apihttp.WithLogger(app.Logger.Logger)) // makes *slog.Logger available to RespondError via context

	// --- Public routes ---
	r.Get("/healthz", healthzHandler)

	// --- Authenticated routes ---
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(auth.RequireAuth(app.Firebase.Auth, app.AuthStore, app.Logger.Logger))

		// registers books handlers
		books.Mount(api, app)

		api.Get("/me", meHandler)
	})

	return r
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	apihttp.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, user)
}
