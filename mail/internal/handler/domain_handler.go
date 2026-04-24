package handler

import (
	"net/http"
	"strconv"

	"auto-hub/mail/internal/service"
	"github.com/gin-gonic/gin"
)

// DomainHandler manages DNS domains (creation, default flag, deletion).
// Only operators may access these routes.
type DomainHandler struct {
	domainService *service.DomainService
}

// NewDomainHandler creates a DomainHandler.
func NewDomainHandler(domainService *service.DomainService) *DomainHandler {
	return &DomainHandler{domainService: domainService}
}

// List renders the domain management page with all domains.
func (h *DomainHandler) List(c *gin.Context) {
	domains, err := h.domainService.ListAll(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}
	user, _ := c.Get("user")
	c.HTML(http.StatusOK, "domains_list.html", gin.H{
		"Title":   "Domains",
		"User":    user,
		"Domains": domains,
	})
}

// New renders the form for adding a new domain.
func (h *DomainHandler) New(c *gin.Context) {
	user, _ := c.Get("user")
	c.HTML(http.StatusOK, "domains_new.html", gin.H{
		"Title": "New Domain",
		"User":  user,
	})
}

// Create validates the submitted domain and persists it.
func (h *DomainHandler) Create(c *gin.Context) {
	domain := c.PostForm("domain")
	makeDefault := c.PostForm("is_default") == "on"

	actorID, _ := actorFromContext(c)

	_, err := h.domainService.Create(c.Request.Context(), actorID, domain, makeDefault)
	if err != nil {
		user, _ := c.Get("user")
		c.HTML(http.StatusBadRequest, "domains_new.html", gin.H{
			"Title": "New Domain",
			"User":  user,
			"Error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/domains")
}

// SetDefault marks a domain as the default for the organisation.
func (h *DomainHandler) SetDefault(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	actorID, _ := actorFromContext(c)

	_ = h.domainService.SetDefault(c.Request.Context(), actorID, id)
	c.Redirect(http.StatusFound, "/domains")
}

// Delete removes a domain.
func (h *DomainHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	actorID, _ := actorFromContext(c)

	_ = h.domainService.Delete(c.Request.Context(), actorID, id)
	c.Redirect(http.StatusFound, "/domains")
}
