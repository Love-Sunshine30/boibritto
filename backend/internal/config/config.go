package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                    string // "dev" | "prod"
	Port                   string
	DatabaseURL            string
	FirebaseProjectID      string
	FirebaseServiceAccount string
	CORSAllowedOrigins     []string
}

// Load reads configuration from the environment. In dev, it first attempts to
// load a .env.dev file (via APP_ENV_FILE, defaulting to ".env.dev") so you
// don't have to `export` everything by hand locally. In prod this is a no-op
// if no such file exists — real env vars are expected to already be set by
// docker-compose's env_file or the process manager.
func Load() (*Config, error) {
	env := getEnv("APP_ENV", "dev")

	if env != "prod" {
		envFile := getEnv("APP_ENV_FILE", ".env."+env)
		if err := godotenv.Load(envFile); err != nil {
			// Not fatal — the vars might already be exported in the shell.
			// Just make it visible instead of failing silently.
			fmt.Fprintf(os.Stderr, "config: no %s file found, relying on exported env vars\n", envFile)
		}
	}

	cfg := &Config{
		Env:                env,
		Port:               getEnv("PORT", "8080"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
	}

	var missing []string
	cfg.DatabaseURL = requireEnv("DATABASE_URL", &missing)
	cfg.FirebaseProjectID = requireEnv("FIREBASE_PROJECT_ID", &missing)
	cfg.FirebaseServiceAccount = requireEnv("FIREBASE_SERVICE_ACCOUNT_JSON", &missing)

	if len(missing) > 0 {
		return nil, fmt.Errorf("config: missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func requireEnv(key string, missing *[]string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		*missing = append(*missing, key)
	}
	return v
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
