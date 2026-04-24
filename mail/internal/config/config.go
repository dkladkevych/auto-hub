package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabasePath          string
	ServerPort            string
	SessionSecret         string
	SessionMaxAge         time.Duration
	RunMigrations         bool
	RunSeed               bool
	OperatorPassword      string        // plain-text fallback for development only
	OperatorPasswordHash  string        // bcrypt hash for production use
	OperatorSessionSecret string
}

// Load reads environment variables (optionally from a .env file) and returns
// a populated Config.  Every value has a safe default so the server can start
// without explicit configuration during local development.
func Load() *Config {
	_ = godotenv.Load()

	maxAgeMins, _ := strconv.Atoi(getEnv("SESSION_MAX_AGE_MINUTES", "1440"))

	return &Config{
		DatabasePath:          getEnv("DATABASE_PATH", "mail.db"),
		ServerPort:            getEnv("SERVER_PORT", "8080"),
		SessionSecret:         getEnv("SESSION_SECRET", "change-me-in-production"),
		SessionMaxAge:         time.Duration(maxAgeMins) * time.Minute,
		RunMigrations:         getEnv("RUN_MIGRATIONS", "true") == "true",
		RunSeed:               getEnv("RUN_SEED", "false") == "true",
		OperatorPassword:      getEnv("OPERATOR_PASSWORD", ""),
		OperatorPasswordHash:  getEnv("OPERATOR_PASSWORD_HASH", ""),
		OperatorSessionSecret: getEnv("OPERATOR_SESSION_SECRET", "operator-change-me"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
