// Package handler contains HTTP handlers for the Gin router.
package handler

import (
	"auto-hub/mail/internal/models"
	"github.com/gin-gonic/gin"
)

// actorFromContext extracts the authenticated user from the Gin context and
// returns their ID and role.  When the context holds a synthetic operator
// user (ID == 0) the role is "operator".
func actorFromContext(c *gin.Context) (actorID int, actorRole string) {
	u, _ := c.Get("user")
	if user, ok := u.(*models.User); ok {
		return user.ID, user.Role
	}
	return 0, ""
}

// CSRFToken returns the current CSRF token stored in the Gin context by the
// middleware.  Handlers should include this value in every HTML form.
func CSRFToken(c *gin.Context) string {
	tok, _ := c.Get("csrf_token")
	if s, ok := tok.(string); ok {
		return s
	}
	return ""
}
