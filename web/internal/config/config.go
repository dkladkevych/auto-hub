package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	SecretKey         string
	AdminPasswordHash string
	AdminPassword     string
	AdminPath         string
	MailServiceURL    string
	InternalAPIToken  string

	DataDir     string
	DBDir       string
	ListingsDir string
	DBPath      string
}

// Load reads environment variables and returns a populated Config.
func Load() *Config {
	_ = godotenv.Load()

	baseDir, _ := os.Getwd()
	dataDir := filepath.Join(baseDir, "data")

	return &Config{
		SecretKey:         getEnv("SECRET_KEY", "fallback_secret"),
		AdminPasswordHash: getEnv("ADMIN_PASSWORD_HASH", ""),
		AdminPassword:     getEnv("ADMIN_PASSWORD", "fallback_password"),
		AdminPath:         getEnv("ADMIN_PATH", "admin"),
		MailServiceURL:    getEnv("MAIL_SERVICE_URL", "http://127.0.0.1:8085"),
		InternalAPIToken:  getEnv("INTERNAL_API_TOKEN", ""),
		DataDir:           dataDir,
		DBDir:             filepath.Join(dataDir, "db"),
		ListingsDir:       filepath.Join(dataDir, "listings"),
		DBPath:            filepath.Join(dataDir, "db", "db.sqlite"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
