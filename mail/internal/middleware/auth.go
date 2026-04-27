// Package middleware provides HTTP middleware for the Gin router:
// session validation, operator token verification, and role-based access.
package middleware

import (
	"net/http"
	"strings"

	"auto-hub/mail/internal/config"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/service"
	"auto-hub/mail/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthRequired ensures the request carries a valid user session cookie.
// On success it sets "session", "user", and "user_role" in the Gin context.
// On failure it redirects to /login and aborts the request.
func AuthRequired(authService *service.AuthService, sessionSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		signed, err := c.Cookie("session_token")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		token, ok := utils.VerifySignedToken(signed, sessionSecret)
		if !ok {
			c.SetCookie("session_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		session, user, err := authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			c.SetCookie("session_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("session", session)
		c.Set("user", user)
		c.Set("user_role", user.Role)
		c.Next()
	}
}

// InternalAPIAuth returns a middleware that validates an Authorization: Bearer
// header against the configured internal API token.  On success it injects a
// synthetic operator user so that downstream code (e.g. actorFromContext and
// audit logging) works unchanged.
func InternalAPIAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.InternalAPIEnabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "internal api disabled"})
			return
		}

		authHeader := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(authHeader, prefix)
		if token == "" || token != cfg.InternalAPIToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("is_operator", true)
		c.Set("user_role", "operator")
		c.Set("user", &models.User{
			ID:       0,
			FullName: "Internal API",
			Role:     "operator",
			Email:    "internal@system",
			IsActive: true,
		})
		c.Next()
	}
}

// AnyAuthRequired accepts either a normal user session or an operator
// authentication cookie.  This middleware is used for routes that both
// operators and regular users can access (e.g. the dashboard and webmail).
// When the operator cookie is present and valid, a synthetic User with
// Role="operator" is injected into the context so that downstream handlers
// do not need special-case logic.
func AnyAuthRequired(authService *service.AuthService, sessionSecret, operatorSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check operator cookie first
		opToken, err := c.Cookie("operator_auth")
		if err == nil && utils.VerifyOperatorToken(operatorSecret, opToken) {
			c.Set("is_operator", true)
			c.Set("user_role", "operator")
			c.Set("user", &models.User{
				ID:       0,
				FullName: "Operator",
				Role:     "operator",
				Email:    "operator@system",
				IsActive: true,
			})
			c.Next()
			return
		}

		// Fall back to normal session auth
		signed, err := c.Cookie("session_token")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		token, ok := utils.VerifySignedToken(signed, sessionSecret)
		if !ok {
			c.SetCookie("session_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		session, user, err := authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			c.SetCookie("session_token", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("session", session)
		c.Set("user", user)
		c.Set("user_role", user.Role)
		c.Next()
	}
}
