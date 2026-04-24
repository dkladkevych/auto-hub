// Package handler exposes HTTP handlers for the mail web application.
package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/service"
	"github.com/gin-gonic/gin"
)

// WebmailHandler handles the webmail UI routes (inbox, compose, read, etc.).
type WebmailHandler struct {
	webmailService *service.WebmailService
}

// NewWebmailHandler creates a WebmailHandler with the required service.
func NewWebmailHandler(webmailService *service.WebmailService) *WebmailHandler {
	return &WebmailHandler{webmailService: webmailService}
}

func (h *WebmailHandler) currentUser(c *gin.Context) *models.User {
	u, _ := c.Get("user")
	if user, ok := u.(*models.User); ok {
		return user
	}
	return nil
}

func (h *WebmailHandler) parseMailboxID(c *gin.Context) (int, error) {
	return strconv.Atoi(c.Param("id"))
}

func (h *WebmailHandler) requireMailboxAccess(c *gin.Context, mailboxID int) bool {
	user := h.currentUser(c)
	if user == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}
	ok, err := h.webmailService.CanAccess(c.Request.Context(), user, mailboxID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return false
	}
	if !ok {
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}
	return true
}

// Inbox displays the message list for a folder (default Inbox).
func (h *WebmailHandler) Inbox(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	folder := c.DefaultQuery("folder", "Inbox")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit

	folders, err := h.webmailService.ListFolders(c.Request.Context(), mailboxID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	messages, err := h.webmailService.ListMessages(c.Request.Context(), mailboxID, folder, limit, offset)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	user := h.currentUser(c)
	var accessibleMailboxes []models.Mailbox
	if user != nil && (user.Role == "admin" || user.Role == "user") {
		accessibleMailboxes, _ = h.webmailService.ListAccessibleMailboxes(c.Request.Context(), user)
	}
	c.HTML(http.StatusOK, "webmail_inbox.html", gin.H{
		"Title":     folder + " — Webmail",
		"User":      user,
		"MailboxID": mailboxID,
		"Folder":    folder,
		"Folders":   folders,
		"Messages":  messages,
		"Page":      page,
		"Mailboxes": accessibleMailboxes,
	})
}

// MessageDetail shows a single message.
func (h *WebmailHandler) MessageDetail(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	messageID := c.Param("messageId")
	folder := c.DefaultQuery("folder", "Inbox")

	msg, err := h.webmailService.GetMessage(c.Request.Context(), mailboxID, folder, messageID)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	// Auto-mark as seen when opened — stealth mode for operator only
	user := h.currentUser(c)
	if !msg.Seen && user != nil && user.Role != "operator" {
		_ = h.webmailService.MarkSeen(c.Request.Context(), mailboxID, folder, messageID, true)
		msg.Seen = true
	}

	var accessibleMailboxes []models.Mailbox
	if user != nil && (user.Role == "admin" || user.Role == "user") {
		accessibleMailboxes, _ = h.webmailService.ListAccessibleMailboxes(c.Request.Context(), user)
	}

	c.HTML(http.StatusOK, "webmail_message.html", gin.H{
		"Title":     msg.Subject,
		"User":      user,
		"MailboxID": mailboxID,
		"Folder":    folder,
		"Message":   msg,
		"HTMLBody":  template.HTML(msg.HTMLBody),
		"Mailboxes": accessibleMailboxes,
	})
}

// ComposePage renders the compose form.
func (h *WebmailHandler) ComposePage(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	user := h.currentUser(c)
	var sendable []models.Mailbox
	if user != nil {
		sendable, _ = h.webmailService.ListSendableMailboxes(c.Request.Context(), user)
	}

	var accessible []models.Mailbox
	if user != nil && (user.Role == "admin" || user.Role == "user") {
		accessible, _ = h.webmailService.ListAccessibleMailboxes(c.Request.Context(), user)
	}

	c.HTML(http.StatusOK, "webmail_compose.html", gin.H{
		"Title":       "Compose",
		"User":        user,
		"MailboxID":   mailboxID,
		"FromOptions": sendable,
		"Mailboxes":   accessible,
		"IsCompose":   true,
	})
}

// Send handles sending a message.
func (h *WebmailHandler) Send(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	// Determine sender mailbox
	fromID := mailboxID
	if fid := c.PostForm("from_mailbox_id"); fid != "" {
		parsed, _ := strconv.Atoi(fid)
		if parsed > 0 {
			fromID = parsed
		}
	}

	// Ensure the user also has access to the chosen sender mailbox
	if fromID != mailboxID {
		if !h.requireMailboxAccess(c, fromID) {
			return
		}
	}

	msg := &models.OutgoingMessage{
		To:       c.PostForm("to"),
		Cc:       c.PostForm("cc"),
		Subject:  c.PostForm("subject"),
		TextBody: c.PostForm("text_body"),
		HTMLBody: c.PostForm("html_body"),
	}

	if msg.To == "" || msg.Subject == "" {
		c.HTML(http.StatusBadRequest, "webmail_compose.html", gin.H{
			"Title":     "Compose",
			"User":      h.currentUser(c),
			"MailboxID": mailboxID,
			"Error":     "To and Subject are required",
			"Draft":     msg,
		})
		return
	}

	if err := h.webmailService.SendMessage(c.Request.Context(), fromID, msg); err != nil {
		c.HTML(http.StatusBadRequest, "webmail_compose.html", gin.H{
			"Title":     "Compose",
			"User":      h.currentUser(c),
			"MailboxID": mailboxID,
			"Error":     err.Error(),
			"Draft":     msg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(fromID)+"/inbox?folder=Sent")
}

// SaveDraft saves a message as draft.
func (h *WebmailHandler) SaveDraft(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	msg := &models.OutgoingMessage{
		To:       c.PostForm("to"),
		Cc:       c.PostForm("cc"),
		Subject:  c.PostForm("subject"),
		TextBody: c.PostForm("text_body"),
		HTMLBody: c.PostForm("html_body"),
	}

	if err := h.webmailService.SaveDraft(c.Request.Context(), mailboxID, msg); err != nil {
		c.HTML(http.StatusBadRequest, "webmail_compose.html", gin.H{
			"Title":     "Compose",
			"User":      h.currentUser(c),
			"MailboxID": mailboxID,
			"Error":     err.Error(),
			"Draft":     msg,
		})
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID)+"/inbox?folder=Drafts")
}

// MyInbox redirects a user to their personal/member inbox.
// Operators go to the admin dashboard instead.
func (h *WebmailHandler) MyInbox(c *gin.Context) {
	user := h.currentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if user.Role == "operator" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	mailboxes, err := h.webmailService.ListAccessibleMailboxes(c.Request.Context(), user)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}
	if len(mailboxes) == 0 {
		c.String(http.StatusNotFound, "No accessible mailboxes")
		return
	}

	// Redirect to the first accessible mailbox inbox
	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxes[0].ID)+"/inbox")
}

// MarkSeen toggles the seen flag for a message.
func (h *WebmailHandler) MarkSeen(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	messageID := c.Param("messageId")
	folder := c.PostForm("folder")
	if folder == "" {
		folder = "Inbox"
	}
	seen := c.PostForm("seen") == "true"

	if err := h.webmailService.MarkSeen(c.Request.Context(), mailboxID, folder, messageID, seen); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID)+"/inbox?folder="+folder)
}

// DeleteMessage moves a message to Trash.
func (h *WebmailHandler) DeleteMessage(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	messageID := c.Param("messageId")
	folder := c.PostForm("folder")
	if folder == "" {
		folder = "Inbox"
	}

	if err := h.webmailService.DeleteMessage(c.Request.Context(), mailboxID, folder, messageID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID)+"/inbox?folder="+folder)
}

// EmptyTrash permanently deletes all messages in the Trash folder.
func (h *WebmailHandler) EmptyTrash(c *gin.Context) {
	mailboxID, err := h.parseMailboxID(c)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid mailbox ID")
		return
	}
	if !h.requireMailboxAccess(c, mailboxID) {
		return
	}

	if err := h.webmailService.EmptyTrash(c.Request.Context(), mailboxID); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/mailboxes/"+strconv.Itoa(mailboxID)+"/inbox?folder=Trash")
}
