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

// List renders the mailbox list page with filters, search and pagination.
func (h *MailboxHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := 20

	filterType := c.Query("type")
	status := c.Query("status")
	search := c.Query("search")

	var active *bool
	if status == "active" {
		v := true
		active = &v
	} else if status == "inactive" {
		v := false
		active = &v
	}

	mailboxes, total, err := h.mailboxService.ListFilteredPaginated(c.Request.Context(), filterType, active, search, page, perPage)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}
	currentUser, _ := c.Get("user")
	u, _ := currentUser.(*models.User)

	// Non-operators never see system mailboxes.
	// For personal mailboxes reflect the user status so that deactivating a
	// user immediately shows as inactive in the list without needing a manual
	// mailbox re-save.
	effectiveActive := make(map[int]bool)
	var filtered []models.Mailbox
	for _, m := range mailboxes {
		if m.MailboxType == "system" && (u == nil || u.Role != "operator") {
			continue
		}
		if m.MailboxType == "personal" {
			usr, _ := h.userService.GetByEmail(c.Request.Context(), m.Email)
			if usr != nil {
				effectiveActive[m.ID] = usr.IsActive
			} else {
				effectiveActive[m.ID] = m.IsActive
			}
		} else {
			effectiveActive[m.ID] = m.IsActive
		}
		filtered = append(filtered, m)
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	c.HTML(http.StatusOK, "mailboxes/list.html", gin.H{
		"CSRFToken": CSRFToken(c),
		"Title":         "Mailboxes",
		"User":          currentUser,
		"Mailboxes":     filtered,
		"EffectiveActive": effectiveActive,
		"Page":          page,
		"TotalPages":    totalPages,
		"HasPrev":       page > 1,
		"HasNext":       page < totalPages,
		"PrevPage":      page - 1,
		"NextPage":      page + 1,
		"FilterType":    filterType,
		"FilterStatus":  status,
		"Search":        search,
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
	c.HTML(http.StatusOK, "mailboxes/new.html", gin.H{
		"CSRFToken": CSRFToken(c),
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
	if mailboxType == "" {
		mailboxType = "shared"
	}
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
			c.HTML(http.StatusBadRequest, "mailboxes/new.html", gin.H{
		"CSRFToken": CSRFToken(c),
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
		c.HTML(http.StatusBadRequest, "mailboxes/new.html", gin.H{
		"CSRFToken": CSRFToken(c),
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

	c.HTML(http.StatusOK, "mailboxes/detail.html", gin.H{
		"CSRFToken": CSRFToken(c),
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

// Delete soft-deletes a mailbox.
func (h *MailboxHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.Delete(c.Request.Context(), actorID, id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes")
}

// Reactivate restores a soft-deleted mailbox.
func (h *MailboxHandler) Reactivate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	actorID, _ := actorFromContext(c)

	if err := h.mailboxService.Reactivate(c.Request.Context(), actorID, id); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(id))
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
