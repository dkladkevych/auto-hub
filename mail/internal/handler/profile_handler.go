// Package handler contains HTTP handlers for the Gin router.
package handler

import (
	"net/http"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/service"
	"github.com/gin-gonic/gin"
)

// ProfileHandler lets authenticated users manage their own profile
// (full name and password).
type ProfileHandler struct {
	userService *service.UserService
}

// NewProfileHandler creates a ProfileHandler.
func NewProfileHandler(userService *service.UserService) *ProfileHandler {
	return &ProfileHandler{userService: userService}
}

// SettingsPage renders the profile settings form.
func (h *ProfileHandler) SettingsPage(c *gin.Context) {
	userVal, _ := c.Get("user")
	user, ok := userVal.(*models.User)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	c.HTML(http.StatusOK, "settings/settings.html", gin.H{
		"CSRFToken": CSRFToken(c),
		"Title": "Settings",
		"User":  user,
	})
}

// UpdateProfile saves changes to the current user's profile.
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userVal, _ := c.Get("user")
	user, ok := userVal.(*models.User)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	fullName := c.PostForm("full_name")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	if newPassword != "" && newPassword != confirmPassword {
		c.HTML(http.StatusBadRequest, "settings/settings.html", gin.H{
		"CSRFToken": CSRFToken(c),
			"Title": "Settings",
			"User":  user,
			"Error": "Passwords do not match",
		})
		return
	}

	if err := h.userService.UpdateProfile(c.Request.Context(), user.ID, fullName, newPassword); err != nil {
		c.HTML(http.StatusBadRequest, "settings/settings.html", gin.H{
		"CSRFToken": CSRFToken(c),
			"Title": "Settings",
			"User":  user,
			"Error": err.Error(),
		})
		return
	}

	// Refresh the name in the context so the nav shows the new name immediately.
	user.FullName = fullName
	c.Redirect(http.StatusFound, "/settings")
}
