package handler

import (
	"net/http"

	"auto-hub/mail/internal/service"
	"github.com/gin-gonic/gin"
)

// InternalHandler exposes JSON endpoints for server-to-server communication.
type InternalHandler struct {
	service *service.InternalAPIService
}

// NewInternalHandler creates a new handler wired to the internal API service.
func NewInternalHandler(service *service.InternalAPIService) *InternalHandler {
	return &InternalHandler{service: service}
}

type sendRequest struct {
	From    string `json:"from" binding:"required,email"`
	To      string `json:"to" binding:"required,email"`
	Subject string `json:"subject" binding:"required"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
}

// Send handles POST /internal/send.
// It accepts a JSON payload, validates that the `from` mailbox is an active
// system sender, and dispatches the email through the mail provider.
func (h *InternalHandler) Send(c *gin.Context) {
	var req sendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Text == "" && req.HTML == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "either text or html must be provided"})
		return
	}

	if err := h.service.Send(c.Request.Context(), req.From, req.To, req.Subject, req.Text, req.HTML); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}
