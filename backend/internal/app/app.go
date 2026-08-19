package app

import (
	"database/sql"

	fbauth "firebase.google.com/go/v4/auth"
	fbmessaging "firebase.google.com/go/v4/messaging"
	"github.com/go-chi/httplog/v2"

	"boibritto/internal/auth"
	"boibritto/internal/config"
)

// App holds every app-wide dependency, constructed once at startup and
// passed down to whatever needs it. Nothing in this codebase should reach
// for a package-level global instead of a field on App.
type App struct {
	Config   *config.Config
	Logger   *httplog.Logger
	DB       *sql.DB
	Firebase FirebaseClients

	// Domain stores/services get added here as each package is built.
	AuthStore *auth.Store
}

type FirebaseClients struct {
	Auth      *fbauth.Client
	Messaging *fbmessaging.Client
}

// New wires together an App from already-initialized dependencies. It
// doesn't do any connecting/initializing itself — that stays in main.go's
// mustX helpers, which know how to fail fast with proper logging if a
// dependency can't be reached. New is pure composition.
func New(cfg *config.Config, logger *httplog.Logger, db *sql.DB, fb FirebaseClients) *App {
	return &App{
		Config:    cfg,
		Logger:    logger,
		DB:        db,
		Firebase:  fb,
		AuthStore: auth.NewStore(db),
	}
}
