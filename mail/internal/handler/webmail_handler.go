// Package handler exposes HTTP handlers for the mail web application.
package handler

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
	"auto-hub/mail/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

// WebmailHandler handles the webmail UI routes (inbox, compose, read, etc.).
type WebmailHandler struct {
	webmailService *service.WebmailService
	maxUploadBytes int64
	uploadDir      string
	auditRepo      *repo.AuditRepo
}

// NewWebmailHandler creates a WebmailHandler with the required service.
func NewWebmailHandler(webmailService *service.WebmailService, maxUploadBytes int64, uploadDir string, auditRepo *repo.AuditRepo) *WebmailHandler {
	return &WebmailHandler{webmailService: webmailService, maxUploadBytes: maxUploadBytes, uploadDir: uploadDir, auditRepo: auditRepo}
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
	// Log operator access to any mailbox (including personal/shared/system)
	if user.Role == "operator" && h.auditRepo != nil {
		_ = h.auditRepo.Log(c.Request.Context(), &models.AuditLog{
			Action:     "operator_mailbox_access",
			EntityType: "mailbox",
			EntityID:   &mailboxID,
			Payload: map[string]interface{}{
				"ip":        c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			},
		})
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

	total, err := h.webmailService.CountMessages(c.Request.Context(), mailboxID, folder)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}
	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	user := h.currentUser(c)
	var accessibleMailboxes []models.Mailbox
	if user != nil && (user.Role == "admin" || user.Role == "user") {
		accessibleMailboxes, _ = h.webmailService.ListAccessibleMailboxes(c.Request.Context(), user)
	}
	c.HTML(http.StatusOK, "webmail/inbox.html", gin.H{
		"CSRFToken": CSRFToken(c),
		"Title":      folder + " — Webmail",
		"User":       user,
		"MailboxID":  mailboxID,
		"Folder":     folder,
		"Folders":    folders,
		"Messages":   messages,
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
		"Mailboxes":  accessibleMailboxes,
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

	policy := bluemonday.UGCPolicy()
	sanitizedHTML := policy.Sanitize(msg.HTMLBody)
	c.HTML(http.StatusOK, "webmail/message.html", gin.H{
		"CSRFToken": CSRFToken(c),
		"Title":     msg.Subject,
		"User":      user,
		"MailboxID": mailboxID,
		"Folder":    folder,
		"Message":   msg,
		"HTMLBody":  template.HTML(sanitizedHTML),
		"Mailboxes": accessibleMailboxes,
		"IsDraft":   folder == "Drafts",
	})
}

// ReplyPage renders the compose form pre-filled for a reply.
func (h *WebmailHandler) ReplyPage(c *gin.Context) {
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

	// Build reply subject
	subject := msg.Subject
	if len(subject) < 3 || subject[:3] != "Re:" {
		subject = "Re: " + subject
	}

	// Build quoted text body
	quoted := fmt.Sprintf("\n\nOn %s, %s wrote:\n> %s",
		msg.Date.Format("Mon, Jan 02, 2006 15:04"),
		msg.From,
		strings.ReplaceAll(msg.TextBody, "\n", "\n> "))

	reply := &models.OutgoingMessage{
		To:        msg.From,
		Subject:   subject,
		TextBody:  quoted,
		InReplyTo: msg.ID,
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

	c.HTML(http.StatusOK, "webmail/compose.html", gin.H{
		"CSRFToken": CSRFToken(c),
		"Title":       "Reply",
		"User":        user,
		"MailboxID":   mailboxID,
		"FromOptions": sendable,
		"Mailboxes":   accessible,
		"IsCompose":   true,
		"Draft":       reply,
		"InReplyTo":   msg.ID,
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

	data := gin.H{
		"CSRFToken": CSRFToken(c),
		"Title":       "Compose",
		"User":        user,
		"MailboxID":   mailboxID,
		"FromOptions": sendable,
		"Mailboxes":   accessible,
		"IsCompose":   true,
	}

	// Load draft if editing
	draftID := c.Query("draft_id")
	if draftID != "" {
		draft, err := h.webmailService.GetMessage(c.Request.Context(), mailboxID, "Drafts", draftID)
		if err == nil && draft != nil {
			data["Draft"] = &models.OutgoingMessage{
				To:        draft.To,
				Cc:        draft.Cc,
				Subject:   draft.Subject,
				TextBody:  draft.TextBody,
				HTMLBody:  draft.HTMLBody,
				InReplyTo: draft.InReplyTo,
			}
			data["DraftID"] = draftID
		}
	}

	c.HTML(http.StatusOK, "webmail/compose.html", data)
}

func (h *WebmailHandler) saveAttachments(c *gin.Context) ([]models.Attachment, error) {
	if h.maxUploadBytes > 0 {
		if err := c.Request.ParseMultipartForm(h.maxUploadBytes); err != nil {
			return nil, fmt.Errorf("upload too large")
		}
	}
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil // no attachments
	}
	files := form.File["attachments"]
	if len(files) == 0 {
		return nil, nil
	}

	uploadDir := h.uploadDir
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, err
	}

	var atts []models.Attachment
	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			continue
		}
		path := filepath.Join(uploadDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fh.Filename)))
		dst, err := os.Create(path)
		if err != nil {
			src.Close()
			continue
		}
		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if err != nil {
			continue
		}
		atts = append(atts, models.Attachment{
			Filename:    fh.Filename,
			ContentType: fh.Header.Get("Content-Type"),
			Size:        fh.Size,
			FilePath:    path,
		})
	}
	return atts, nil
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

	atts, _ := h.saveAttachments(c)

	msg := &models.OutgoingMessage{
		To:          c.PostForm("to"),
		Cc:          c.PostForm("cc"),
		Subject:     c.PostForm("subject"),
		TextBody:    c.PostForm("text_body"),
		HTMLBody:    c.PostForm("html_body"),
		InReplyTo:   c.PostForm("in_reply_to"),
		Attachments: atts,
	}

	// Check send permission
	mb, err := h.webmailService.GetMailboxByID(c.Request.Context(), fromID)
	if err != nil || mb == nil {
		c.String(http.StatusInternalServerError, "Error: mailbox not found")
		return
	}
	if !mb.CanSend {
		user := h.currentUser(c)
		var sendable []models.Mailbox
		if user != nil {
			sendable, _ = h.webmailService.ListSendableMailboxes(c.Request.Context(), user)
		}
		var accessible []models.Mailbox
		if user != nil && (user.Role == "admin" || user.Role == "user") {
			accessible, _ = h.webmailService.ListAccessibleMailboxes(c.Request.Context(), user)
		}
		c.HTML(http.StatusOK, "webmail/compose.html", gin.H{
			"CSRFToken":   CSRFToken(c),
			"Title":       "Compose",
			"User":        user,
			"MailboxID":   mailboxID,
			"FromOptions": sendable,
			"Mailboxes":   accessible,
			"IsCompose":   true,
			"Draft":       msg,
			"SendBlocked": true,
		})
		return
	}

	if msg.To == "" || msg.Subject == "" {
		c.HTML(http.StatusBadRequest, "webmail/compose.html", gin.H{
		"CSRFToken": CSRFToken(c),
			"Title":     "Compose",
			"User":      h.currentUser(c),
			"MailboxID": mailboxID,
			"Error":     "To and Subject are required",
			"Draft":     msg,
		})
		return
	}

	if err := h.webmailService.SendMessage(c.Request.Context(), fromID, msg); err != nil {
		c.HTML(http.StatusBadRequest, "webmail/compose.html", gin.H{
		"CSRFToken": CSRFToken(c),
			"Title":     "Compose",
			"User":      h.currentUser(c),
			"MailboxID": mailboxID,
			"Error":     err.Error(),
			"Draft":     msg,
		})
		return
	}

	// Audit log outbound send
	if h.auditRepo != nil {
		user := h.currentUser(c)
		var actorID *int
		if user != nil && user.ID != 0 {
			actorID = &user.ID
		}
		_ = h.auditRepo.Log(c.Request.Context(), &models.AuditLog{
			ActorUserID: actorID,
			Action:      "outbound_send",
			EntityType:  "message",
			Payload: map[string]interface{}{
				"from":            mb.Email,
				"to":              msg.To,
				"cc":              msg.Cc,
				"subject":         msg.Subject,
				"has_attachments": len(msg.Attachments) > 0,
				"ip":              c.ClientIP(),
			},
		})
	}

	// Remove draft after successful send
	draftID := c.PostForm("draft_id")
	if draftID != "" {
		_ = h.webmailService.DeleteMessage(c.Request.Context(), mailboxID, "Drafts", draftID)
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

	atts, _ := h.saveAttachments(c)

	msg := &models.OutgoingMessage{
		To:          c.PostForm("to"),
		Cc:          c.PostForm("cc"),
		Subject:     c.PostForm("subject"),
		TextBody:    c.PostForm("text_body"),
		HTMLBody:    c.PostForm("html_body"),
		Attachments: atts,
	}

	if err := h.webmailService.SaveDraft(c.Request.Context(), mailboxID, msg); err != nil {
		c.HTML(http.StatusBadRequest, "webmail/compose.html", gin.H{
		"CSRFToken": CSRFToken(c),
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

// ToggleFlag toggles the starred (flagged) state of a message.
func (h *WebmailHandler) ToggleFlag(c *gin.Context) {
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
	flagged := c.PostForm("flagged") == "true"

	if err := h.webmailService.SetFlagged(c.Request.Context(), mailboxID, folder, messageID, flagged); err != nil {
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
