// Package handler exposes HTTP handlers for the mail web application.
package handler

import (
	"net/http"
	"strconv"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/service"
	"github.com/gin-gonic/gin"
)

// MailboxHandler handles mailbox CRUD and member management routes.
type MailboxHandler struct {
	mailboxService *service.MailboxService
	userService    *service.UserService
	domainService  *service.DomainService
}

// NewMailboxHandler creates a MailboxHandler with the required services.
func NewMailboxHandler(mailboxService *service.MailboxService, userService *service.UserService, domainService *service.DomainService) *MailboxHandler {
	return &MailboxHandler{
		mailboxService: mailboxService,
		userService:    userService,
		domainService:  domainService,
	}
}

// List renders the mailbox list page. System mailboxes are hidden from non-operators.
func (h *MailboxHandler) List(c *gin.Context) {
	mailboxes, err := h.mailboxService.List(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}
	currentUser, _ := c.Get("user")
	u, _ := currentUser.(*models.User)

	var filtered []models.Mailbox
	for _, m := range mailboxes {
		if m.MailboxType == "system" && (u == nil || u.Role != "operator") {
			continue
		}
		filtered = append(filtered, m)
	}

	c.HTML(http.StatusOK, "mailboxes_list.html", gin.H{
		"Title":     "Mailboxes",
		"User":      currentUser,
		"Mailboxes": filtered,
	})
}

// New renders the new-mailbox form.
func (h *MailboxHandler) New(c *gin.Context) {
	domains, _ := h.domainService.ListActive(c.Request.Context())
	user, _ := c.Get("user")
	canSetSystemType := false
	if u, ok := user.(*models.User); ok && u.Role == "operator" {
		canSetSystemType = true
	}
	preselectType := c.Query("type")
	if preselectType != "shared" && preselectType != "system" {
		preselectType = "shared"
	}
	c.HTML(http.StatusOK, "mailboxes_new.html", gin.H{
		"Title":            "New Mailbox",
		"User":             user,
		"Domains":          domains,
		"CanSetSystemType": canSetSystemType,
		"PreselectType":    preselectType,
	})
}

// Create persists a new mailbox.
func (h *MailboxHandler) Create(c *gin.Context) {
	localPart := c.PostForm("local_part")
	domain := c.PostForm("domain")
	displayName := c.PostForm("display_name")
	mailboxType := c.PostForm("mailbox_type")
	canReceive := c.PostForm("can_receive") == "on"
	canSend := c.PostForm("can_send") == "on"
	quotaMb, _ := strconv.Atoi(c.PostForm("quota_mb"))
	if quotaMb <= 0 {
		quotaMb = 1024
	}
	password := c.PostForm("password")

	if domain == "" {
		defaultDomain, err := h.domainService.GetDefaultDomain(c.Request.Context())
		if err != nil {
			domains, _ := h.domainService.ListActive(c.Request.Context())
			user, _ := c.Get("user")
			c.HTML(http.StatusBadRequest, "mailboxes_new.html", gin.H{
				"Title":   "New Mailbox",
				"User":    user,
				"Domains": domains,
				"Error":   "No domain selected and no default domain configured",
			})
			return
		}
		domain = defaultDomain
	}

	actorID, actorRole := actorFromContext(c)
	if actorRole == "admin" && mailboxType == "system" {
		mailboxType = "shared"
	}

	_, err := h.mailboxService.Create(c.Request.Context(), actorID, localPart, domain, displayName, mailboxType, canReceive, canSend, quotaMb, password)
	if err != nil {
		domains, _ := h.domainService.ListActive(c.Request.Context())
		user, _ := c.Get("user")
		c.HTML(http.StatusBadRequest, "mailboxes_new.html", gin.H{
			"Title":   "New Mailbox",
			"User":    user,
			"Domains": domains,
			"Error":   err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes")
}

// Detail renders the mailbox detail page with members and associated user.
func (h *MailboxHandler) Detail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid ID")
		return
	}

	m, members, err := h.mailboxService.GetWithMembers(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	users, _ := h.userService.List(c.Request.Context())
	currentUser, _ := c.Get("user")

	var editUser *models.User
	if m.MailboxType == "personal" {
		editUser, _ = h.userService.GetByEmail(c.Request.Context(), m.Email)
	}

	canSetRole := false
	if u, ok := currentUser.(*models.User); ok && u.Role == "operator" {
		canSetRole = true
	}

	c.HTML(http.StatusOK, "mailboxes_detail.html", gin.H{
		"Title":      m.Email,
		"User":       currentUser,
		"Mailbox":    m,
		"Members":    members,
		"Users":      users,
		"EditUser":   editUser,
		"CanSetRole": canSetRole,
	})
}

// Update modifies an existing mailbox.
func (h *MailboxHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	displayName := c.PostForm("display_name")
	mailboxType := c.PostForm("mailbox_type")
	quotaMb, _ := strconv.Atoi(c.PostForm("quota_mb"))
	if quotaMb <= 0 {
		quotaMb = 1024
	}

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.Update(c.Request.Context(), actorID, id, displayName, mailboxType, quotaMb); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(id))
}

// Delete removes a mailbox.
func (h *MailboxHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.Delete(c.Request.Context(), actorID, id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes")
}

// AddMember adds a user to a mailbox.
func (h *MailboxHandler) AddMember(c *gin.Context) {
	mailboxID, _ := strconv.Atoi(c.Param("id"))
	userID, _ := strconv.Atoi(c.PostForm("user_id"))
	accessRole := c.PostForm("access_role")

	actor, _ := c.Get("user")
	actorID := 0
	if u, ok := actor.(*models.User); ok {
		actorID = u.ID
	}

	_, err := h.mailboxService.AddMember(c.Request.Context(), actorID, mailboxID, userID, accessRole)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID))
}

// RemoveMember revokes a user's access to a mailbox.
func (h *MailboxHandler) RemoveMember(c *gin.Context) {
	mailboxID := c.Param("id")
	memberID, _ := strconv.Atoi(c.Param("memberId"))

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.RemoveMember(c.Request.Context(), actorID, memberID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+mailboxID)
}

// UpdateSettings toggles receive/send capabilities for a mailbox.
func (h *MailboxHandler) UpdateSettings(c *gin.Context) {
	mailboxID, _ := strconv.Atoi(c.Param("id"))
	canReceive := c.PostForm("can_receive") == "on"
	canSend := c.PostForm("can_send") == "on"

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.UpdateSettings(c.Request.Context(), actorID, mailboxID, canReceive, canSend); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID))
}

// SetPassword sets a new mailbox-level password.
func (h *MailboxHandler) SetPassword(c *gin.Context) {
	mailboxID, _ := strconv.Atoi(c.Param("id"))
	password := c.PostForm("password")

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.SetPassword(c.Request.Context(), actorID, mailboxID, password); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID))
}

// ResetPassword clears the mailbox-level password.
func (h *MailboxHandler) ResetPassword(c *gin.Context) {
	mailboxID, _ := strconv.Atoi(c.Param("id"))

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.ResetPassword(c.Request.Context(), actorID, mailboxID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID))
}
