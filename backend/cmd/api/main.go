package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"boibritto/internal/app"
	"boibritto/internal/config"
	"boibritto/internal/platform/firebase"
	"boibritto/internal/platform/httpserver"
	"boibritto/internal/platform/logging"
	"boibritto/internal/platform/postgres"
)

func main() {

	// Global context for App
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Loads config
	cfg := mustLoadConfig()

	// Logger initialized
	logger := logging.New(cfg.Env)

	// Firebase client initialized
	fbClients := mustInitFirebase(ctx, cfg, logger.Logger)

	// Connects database
	db := mustConnectDB(ctx, cfg, logger.Logger)

	// app holds all app-wide dependencies
	app := app.New(cfg, logger, db, app.FirebaseClients{
		Auth:      fbClients.Auth,
		Messaging: fbClients.Messaging,
	})

	// Chi router
	router := httpserver.NewRouter(app)

	// HTTP server
	server := httpserver.NewServer("127.0.0.1:"+cfg.Port, router)

	// Run blocks until the server exits — either an unexpected error, or a
	// clean shutdown triggered by ctx being canceled (signal received).
	if err := httpserver.Run(ctx, server, logger.Logger); err != nil {
		logger.Logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}

	logger.Info("server shut down cleanly")
}

func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		// Logger doesn't exist yet at this point — this is the one place in
		// the app allowed to bypass structured logging entirely.
		panic(err)
	}
	return cfg
}

func mustInitFirebase(ctx context.Context, cfg *config.Config, logger *slog.Logger) *firebase.Clients {
	fbClients, err := firebase.New(ctx, cfg.FirebaseProjectID, cfg.FirebaseServiceAccount)
	if err != nil {
		logger.Error("failed to initialize firebase", "error", err)
		os.Exit(1)
	}
	logger.Info("firebase initialized", "project_id", cfg.FirebaseProjectID)
	return fbClients
}

func mustConnectDB(ctx context.Context, cfg *config.Config, logger *slog.Logger) *sql.DB {
	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	logger.Info("database connected")

	if err := postgres.Migrate(db); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	return db
}
