// Package handler contains HTTP handlers for the Gin router.  Each handler
// is a thin layer that extracts request data, delegates to the appropriate
// service, and renders HTML templates or redirects.
package handler

import (
	"net/http"

	"auto-hub/mail/internal/config"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/service"
	"auto-hub/mail/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthHandler manages login, logout and session cookie handling for regular
// users (admins and standard users).  Operator authentication is handled
// separately by OperatorHandler.
type AuthHandler struct {
	authService   *service.AuthService
	domainService *service.DomainService
	cfg           *config.Config
}

// NewAuthHandler creates an AuthHandler with the required services.
func NewAuthHandler(authService *service.AuthService, domainService *service.DomainService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{authService: authService, domainService: domainService, cfg: cfg}
}

// LoginPage renders the login form with a list of active domains.
func (h *AuthHandler) LoginPage(c *gin.Context) {
	domains, _ := h.domainService.ListActive(c.Request.Context())
	c.HTML(http.StatusOK, "auth/login.html", gin.H{
		"CSRFToken": CSRFToken(c),
		"Title":   "Login",
		"Domains": domains,
	})
}

// Login validates the submitted credentials, creates a session, writes the
// session cookie and redirects to the application root.
func (h *AuthHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	domain := c.PostForm("domain")
	password := c.PostForm("password")

	if domain == "" {
		defaultDomain, err := h.domainService.GetDefaultDomain(c.Request.Context())
		if err != nil {
			domains, _ := h.domainService.ListActive(c.Request.Context())
			c.HTML(http.StatusUnauthorized, "auth/login.html", gin.H{
		"CSRFToken": CSRFToken(c),
				"Title":   "Login",
				"Domains": domains,
				"Error":   "No domain selected and no default domain configured",
			})
			return
		}
		domain = defaultDomain
	}

	email := username + "@" + domain

	_, token, err := h.authService.Login(c.Request.Context(), email, password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		domains, _ := h.domainService.ListActive(c.Request.Context())
		c.HTML(http.StatusUnauthorized, "auth/login.html", gin.H{
		"CSRFToken": CSRFToken(c),
			"Title":   "Login",
			"Domains": domains,
			"Error":   "Invalid username or password",
		})
		return
	}

	// Sign the token with HMAC so the cookie cannot be forged.
	signed := token + "." + utils.SignToken(token, h.cfg.SessionSecret)
	c.SetSameSite(h.cfg.SessionCookieSameSite)
	c.SetCookie("session_token", signed, int(h.cfg.SessionMaxAge.Seconds()), "/", "", h.cfg.SessionCookieSecure, true)
	c.Redirect(http.StatusFound, "/")
}

// Logout destroys the current session and clears the session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionVal, exists := c.Get("session")
	if exists {
		if s, ok := sessionVal.(*models.Session); ok {
			_ = h.authService.Logout(c.Request.Context(), s.ID)
		}
	}

	c.SetSameSite(h.cfg.SessionCookieSameSite)
	c.SetCookie("session_token", "", -1, "/", "", h.cfg.SessionCookieSecure, true)
	c.Redirect(http.StatusFound, "/login")
}
