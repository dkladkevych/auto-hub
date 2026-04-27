package handler

import (
	"fmt"
	"net/http"

	"auto-hub/web/internal/config"
	"auto-hub/web/internal/repo"
	"auto-hub/web/internal/service"
	"github.com/gin-gonic/gin"
)

// PagesHandler handles static pages.
type PagesHandler struct {
	cfg            *config.Config
	listingService *service.ListingService
}

// NewPagesHandler creates a new PagesHandler.
func NewPagesHandler(cfg *config.Config, listingService *service.ListingService) *PagesHandler {
	return &PagesHandler{cfg: cfg, listingService: listingService}
}

// Terms renders the terms page.
func (h *PagesHandler) Terms(c *gin.Context) {
	c.HTML(http.StatusOK, "public_terms", BaseData(c))
}

// Privacy renders the privacy page.
func (h *PagesHandler) Privacy(c *gin.Context) {
	c.HTML(http.StatusOK, "public_privacy", BaseData(c))
}

// Robots serves robots.txt.
func (h *PagesHandler) Robots(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK,
		"User-agent: *\nDisallow: /%s/\nSitemap: /sitemap.xml\n",
		h.cfg.AdminPath,
	)
}

// Sitemap serves sitemap.xml.
func (h *PagesHandler) Sitemap(c *gin.Context) {
	listings, _, _, _, err := h.listingService.HomeListings(repo.FilterParams{}, 1, 10000)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Header("Content-Type", "application/xml")
	c.HTML(http.StatusOK, "sitemap", gin.H{
		"Host":     fmt.Sprintf("%s://%s", c.Request.URL.Scheme, c.Request.Host),
		"Listings": listings,
	})
}
