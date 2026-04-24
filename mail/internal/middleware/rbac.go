// Package middleware (rbac.go) implements role-based access control on
// top of Gin.  It is intended to be chained after AuthRequired or
// AnyAuthRequired so that the "user" context key is already populated.
package middleware

import (
	"net/http"

	"auto-hub/mail/internal/models"
	"github.com/gin-gonic/gin"
)

// RequireRoles returns a middleware that allows the request to proceed
// only when the authenticated user has one of the supplied roles.
// Operators always bypass this check because they are granted global
// access to management routes.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Operator bypasses all role checks
		if isOp, _ := c.Get("is_operator"); isOp != nil && isOp.(bool) {
			c.Next()
			return
		}

		userVal, exists := c.Get("user")
		if !exists {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		user, ok := userVal.(*models.User)
		if !ok {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		for _, role := range allowedRoles {
			if user.Role == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}
