package handler

import (
	"net/http"
	"strconv"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/service"
	"github.com/gin-gonic/gin"
)

// UserHandler manages user creation and updates.  Deletion and listing
// have been moved into the mailbox management UI, but the underlying
// service endpoints remain here.
type UserHandler struct {
	userService   *service.UserService
	domainService *service.DomainService
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(userService *service.UserService, domainService *service.DomainService) *UserHandler {
	return &UserHandler{
		userService:   userService,
		domainService: domainService,
	}
}

// New renders the user creation form.
func (h *UserHandler) New(c *gin.Context) {
	domains, _ := h.domainService.ListActive(c.Request.Context())
	user, _ := c.Get("user")
	canSetRole := false
	if u, ok := user.(*models.User); ok && u.Role == "operator" {
		canSetRole = true
	}
	c.HTML(http.StatusOK, "users_new.html", gin.H{
		"Title":      "New User",
		"User":       user,
		"Domains":    domains,
		"CanSetRole": canSetRole,
	})
}

// Create validates the submitted data and delegates to UserService.Create.
func (h *UserHandler) Create(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	fullName := c.PostForm("full_name")
	role := c.PostForm("role")
	domain := c.PostForm("domain")
	quotaMb, _ := strconv.Atoi(c.PostForm("quota_mb"))
	if quotaMb <= 0 {
		quotaMb = 1024
	}
	canReceive := c.PostForm("can_receive") == "on"
	canSend := c.PostForm("can_send") == "on"

	if domain == "" {
		defaultDomain, err := h.domainService.GetDefaultDomain(c.Request.Context())
		if err != nil {
			user, _ := c.Get("user")
			c.HTML(http.StatusBadRequest, "users_new.html", gin.H{
				"Title": "New User",
				"User":  user,
				"Error": "No domain selected and no default domain configured",
			})
			return
		}
		domain = defaultDomain
	}

	actorID, actorRole := actorFromContext(c)
	if actorRole == "admin" {
		role = "user"
	}

	_, _, err := h.userService.Create(c.Request.Context(), actorID, username, password, fullName, role, domain, quotaMb, canReceive, canSend)
	if err != nil {
		domains, _ := h.domainService.ListActive(c.Request.Context())
		user, _ := c.Get("user")
		c.HTML(http.StatusBadRequest, "users_new.html", gin.H{
			"Title":   "New User",
			"User":    user,
			"Domains": domains,
			"Error":   err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes")
}

// Update modifies an existing user and synchronises the changes to their
// personal mailbox.
func (h *UserHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	fullName := c.PostForm("full_name")
	role := c.PostForm("role")
	isActive := c.PostForm("is_active") == "on"
	quotaMb, _ := strconv.Atoi(c.PostForm("quota_mb"))
	if quotaMb <= 0 {
		quotaMb = 1024
	}
	canReceive := c.PostForm("can_receive") == "on"
	canSend := c.PostForm("can_send") == "on"
	newPassword := c.PostForm("new_password")

	actorID, actorRole := actorFromContext(c)

	// Admin cannot change roles — preserve the current one
	if actorRole == "admin" {
		current, _ := h.userService.GetByID(c.Request.Context(), id)
		if current != nil {
			role = current.Role
		}
	}

	if err := h.userService.Update(c.Request.Context(), actorID, id, fullName, role, isActive, quotaMb, canReceive, canSend, newPassword); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes")
}

// Delete removes a user and their personal mailbox.
func (h *UserHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	actorID, _ := actorFromContext(c)

	if err := h.userService.Delete(c.Request.Context(), actorID, id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes")
}
