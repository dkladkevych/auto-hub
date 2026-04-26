// Package middleware provides HTTP middleware for the Gin router:
// session validation, operator token verification, and role-based access.
package middleware

import (
	"net/http"

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
