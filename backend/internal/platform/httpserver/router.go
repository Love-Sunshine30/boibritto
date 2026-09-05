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
	"boibritto/internal/profile"
	"boibritto/internal/push"
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

	// initializing FCM push notification
	pushStore := push.NewStore(app.DB)
	pushSender := push.NewFCMSender(app.Firebase.Messaging, pushStore, app.Logger.Logger)
	notifier := push.NewRequestNotifier(pushSender)

	// profile activity tracker
	activityTracker := profile.NewStore(app.DB)

	// profile checker checks if profile has whatsApp number and name before a user can list book or make request
	profileChecker := profile.NewStore(app.DB)

	// --- Authenticated routes ---
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(auth.RequireAuth(app.Firebase.Auth, app.AuthStore, activityTracker, app.Logger.Logger))

		// registers books handlers
		books.Mount(api, app, profileChecker)

		// registers requests handlers
		requests.Mount(api, app, profileChecker, notifier)

		// registers push handlers
		push.Mount(api, app, pushStore)

		// registers profile handlers
		profile.Mount(api, app)
	})

	return r
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	apihttp.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
