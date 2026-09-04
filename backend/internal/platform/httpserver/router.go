package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httplog/v2"

	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
	"boibritto/internal/books"
	"boibritto/internal/requests"
)

func NewRouter(app *app.App) chi.Router {
	r := chi.NewRouter()

	// --- Global, unauthenticated middleware ---
	r.Use(middleware.RequestID)
	r.Use(httplog.RequestLogger(app.Logger))
	r.Use(middleware.Recoverer)
	r.Use(apihttp.WithLogger(app.Logger.Logger)) // makes *slog.Logger available to RespondError via context

	// CORS must run before any route-specific middleware (like RequireAuth),
	// and it must handle OPTIONS preflight requests itself — it does, by
	// design: chi's cors middleware intercepts OPTIONS and responds
	// directly, never calling next.ServeHTTP for a preflight request.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.Config.CORSAllowedOrigins, // from config.go, e.g. ["http://localhost:43875"]
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Public routes ---
	r.Get("/healthz", healthzHandler)

	// --- Authenticated routes ---
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(auth.RequireAuth(app.Firebase.Auth, app.AuthStore, app.Logger.Logger))

		// registers books handlers
		books.Mount(api, app)

		// registers requests handlers
		requests.Mount(api, app)

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
