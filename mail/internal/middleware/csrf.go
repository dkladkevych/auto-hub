// Package middleware provides CSRF protection using the double-submit cookie
// pattern.  A signed token is stored in the "csrf_token" cookie and must be
// echoed back in either the _csrf form field or the X-CSRF-Token header.
package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"

	"auto-hub/mail/internal/config"
	"github.com/gin-gonic/gin"
)

const csrfCookieName = "csrf_token"
const csrfContextKey = "csrf_token"
const csrfHeaderName = "X-CSRF-Token"
const csrfFormField = "_csrf"

// generateRandomString returns a URL-safe base64-encoded random string.
func generateRandomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// signCSRF creates an HMAC-SHA256 signature for a raw token.
func signCSRF(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

// verifyCSRF checks that signature matches token and secret.
func verifyCSRF(token, signature, secret string) bool {
	expected := signCSRF(token, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// CSRFProtect returns a Gin middleware that generates and validates CSRF
// tokens.  GET and HEAD requests receive a fresh token; all mutating methods
// must echo it back.
func CSRFProtect(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Safe methods: ensure a fresh CSRF cookie exists.
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			raw := generateRandomString(32)
			signed := raw + "." + signCSRF(raw, cfg.CSRFSecret)
			c.SetSameSite(cfg.SessionCookieSameSite)
			c.SetCookie(csrfCookieName, signed, int(cfg.SessionMaxAge.Seconds()), "/", "", cfg.SessionCookieSecure, false)
			c.Set(csrfContextKey, signed)
			c.Next()
			return
		}

		// Unsafe methods: validate token.
		cookieVal, err := c.Cookie(csrfCookieName)
		if err != nil || cookieVal == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			return
		}

		sent := c.PostForm(csrfFormField)
		if sent == "" {
			sent = c.GetHeader(csrfHeaderName)
		}
		if sent == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			return
		}

		if !compareCSRFTokens(cookieVal, sent, cfg.CSRFSecret) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token invalid"})
			return
		}

		c.Set(csrfContextKey, cookieVal)
		c.Next()
	}
}

// compareCSRFTokens verifies that two CSRF tokens are valid and match.
func compareCSRFTokens(a, b, secret string) bool {
	if a == "" || b == "" {
		return false
	}
	if !verifyCSRFToken(a, secret) || !verifyCSRFToken(b, secret) {
		return false
	}
	return a == b
}

// verifyCSRFToken checks the HMAC signature inside a token.
func verifyCSRFToken(token, secret string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	return verifyCSRF(parts[0], parts[1], secret)
}
