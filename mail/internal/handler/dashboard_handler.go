package handler

import (
	"net/http"

	"auto-hub/mail/internal/models"
	"github.com/gin-gonic/gin"
)

// DashboardHandler renders the landing page after authentication.
type DashboardHandler struct{}

// NewDashboardHandler creates a DashboardHandler.
func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// Index shows the operator dashboard or redirects regular users to their
// inbox because they do not need a management landing page.
func (h *DashboardHandler) Index(c *gin.Context) {
	userVal, _ := c.Get("user")
	if user, ok := userVal.(*models.User); ok && user.Role == "operator" {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"Title": "Dashboard",
			"User":  user,
		})
		return
	}
	// Everyone else (admin, user) goes straight to inbox
	c.Redirect(http.StatusFound, "/inbox")
}
