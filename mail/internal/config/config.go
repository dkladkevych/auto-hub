package config

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabasePath          string
	ServerPort            string
	SessionSecret         string
	SessionMaxAge         time.Duration
	SessionCookieSecure   bool
	SessionCookieSameSite http.SameSite
	CSRFSecret            string
	MailProvider          string
	MaxUploadSizeMB       int
	UploadDir             string
	RunMigrations         bool
	RunSeed               bool
	OperatorPassword      string
	OperatorPasswordHash  string
	OperatorSessionSecret string
	GinMode               string

	// Database driver
	DBDriver    string
	DatabaseURL string

	// SMTP outbound settings
	SMTPEnabled    bool
	SMTPHost       string
	SMTPPort       string
	SMTPRequireTLS bool

	// IMAP settings
	IMAPHost           string
	IMAPPort           string
	IMAPUseSSL         bool
	IMAPSkipTLSVerify  bool
	IMAPMasterPassword string

	// Operator login path
	OperatorLoginPath  string
	OperatorLogoutPath string

	// Rate limiting
	RateLimitEnabled            bool
	LoginRateLimitMax           int
	LoginRateLimitWindow        time.Duration
	OperatorRateLimitMax        int
	OperatorRateLimitWindow     time.Duration
	SendRateLimitMax            int
	SendRateLimitWindow         time.Duration
	DraftRateLimitMax           int
	DraftRateLimitWindow        time.Duration
}

// Load reads environment variables (optionally from a .env file) and returns
// a populated Config.  Every value has a safe default so the server can start
// without explicit configuration during local development.
func Load() *Config {
	_ = godotenv.Load()

	maxAgeMins, _ := strconv.Atoi(getEnv("SESSION_MAX_AGE_MINUTES", "1440"))
	maxUploadMB, _ := strconv.Atoi(getEnv("MAX_UPLOAD_SIZE_MB", "10"))
	if maxUploadMB < 1 {
		maxUploadMB = 10
	}

	operatorLoginPath := getEnv("OPERATOR_LOGIN_PATH", "/operator/login")
	operatorLogoutPath := getEnv("OPERATOR_LOGOUT_PATH", operatorLoginPath+"/logout")

	loginRateLimitMax, _ := strconv.Atoi(getEnv("LOGIN_RATE_LIMIT_MAX", "5"))
	loginRateLimitWindow, _ := strconv.Atoi(getEnv("LOGIN_RATE_LIMIT_WINDOW_MINUTES", "10"))
	opRateLimitMax, _ := strconv.Atoi(getEnv("OPERATOR_RATE_LIMIT_MAX", "3"))
	opRateLimitWindow, _ := strconv.Atoi(getEnv("OPERATOR_RATE_LIMIT_WINDOW_MINUTES", "15"))
	sendRateLimitMax, _ := strconv.Atoi(getEnv("SEND_RATE_LIMIT_MAX", "20"))
	sendRateLimitWindow, _ := strconv.Atoi(getEnv("SEND_RATE_LIMIT_WINDOW_MINUTES", "10"))
	draftRateLimitMax, _ := strconv.Atoi(getEnv("DRAFT_RATE_LIMIT_MAX", "60"))
	draftRateLimitWindow, _ := strconv.Atoi(getEnv("DRAFT_RATE_LIMIT_WINDOW_MINUTES", "10"))

	return &Config{
		DatabasePath:          getEnv("DATABASE_PATH", "data/mail.db"),
		ServerPort:            getEnv("SERVER_PORT", "8080"),
		SessionSecret:         getEnv("SESSION_SECRET", "change-me-in-production"),
		SessionMaxAge:         time.Duration(maxAgeMins) * time.Minute,
		SessionCookieSecure:   getEnv("SESSION_COOKIE_SECURE", "false") == "true",
		SessionCookieSameSite: parseSameSite(getEnv("SESSION_COOKIE_SAMESITE", "Lax")),
		CSRFSecret:            getEnv("CSRF_SECRET", getEnv("SESSION_SECRET", "change-me-in-production")),
		MailProvider:          getEnv("MAIL_PROVIDER", "dev_db"),
		MaxUploadSizeMB:       maxUploadMB,
		UploadDir:             getEnv("UPLOAD_DIR", "./uploads"),
		RunMigrations:         getEnv("RUN_MIGRATIONS", "true") == "true",
		RunSeed:               getEnv("RUN_SEED", "false") == "true",
		OperatorPassword:      getEnv("OPERATOR_PASSWORD", ""),
		OperatorPasswordHash:  getEnv("OPERATOR_PASSWORD_HASH", ""),
		OperatorSessionSecret: getEnv("OPERATOR_SESSION_SECRET", "operator-change-me"),
		GinMode:               getEnv("GIN_MODE", "debug"),

		DBDriver:    getEnv("DB_DRIVER", "sqlite"),
		DatabaseURL: getEnv("DATABASE_URL", ""),

		SMTPEnabled:    getEnv("SMTP_ENABLED", "false") == "true",
		SMTPHost:       getEnv("SMTP_HOST", "127.0.0.1"),
		SMTPPort:       getEnv("SMTP_PORT", "25"),
		SMTPRequireTLS: getEnv("SMTP_REQUIRE_TLS", "false") == "true",

		IMAPHost:           getEnv("IMAP_HOST", "127.0.0.1"),
		IMAPPort:           getEnv("IMAP_PORT", "143"),
		IMAPUseSSL:         getEnv("IMAP_USE_SSL", "false") == "true",
		IMAPSkipTLSVerify:  getEnv("IMAP_SKIP_TLS_VERIFY", "true") == "true",
		IMAPMasterPassword: getEnv("IMAP_MASTER_PASSWORD", ""),

		OperatorLoginPath:  operatorLoginPath,
		OperatorLogoutPath: operatorLogoutPath,

		RateLimitEnabled:        getEnv("RATE_LIMIT_ENABLED", "true") == "true",
		LoginRateLimitMax:       loginRateLimitMax,
		LoginRateLimitWindow:    time.Duration(loginRateLimitWindow) * time.Minute,
		OperatorRateLimitMax:    opRateLimitMax,
		OperatorRateLimitWindow: time.Duration(opRateLimitWindow) * time.Minute,
		SendRateLimitMax:        sendRateLimitMax,
		SendRateLimitWindow:     time.Duration(sendRateLimitWindow) * time.Minute,
		DraftRateLimitMax:       draftRateLimitMax,
		DraftRateLimitWindow:    time.Duration(draftRateLimitWindow) * time.Minute,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseSameSite(v string) http.SameSite {
	switch strings.ToLower(v) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
