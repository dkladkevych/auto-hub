package handler

import (
	"net/http"

	"auto-hub/mail/internal/config"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
	"auto-hub/mail/internal/utils"
	"github.com/gin-gonic/gin"
)

// OperatorHandler manages the super-admin login flow.  The operator is a
// special account that exists outside of the users table and is
// authenticated via a standalone cookie with an HMAC token.
type OperatorHandler struct {
	cfg       *config.Config
	auditRepo *repo.AuditRepo
}

// NewOperatorHandler creates an OperatorHandler.
func NewOperatorHandler(cfg *config.Config, auditRepo *repo.AuditRepo) *OperatorHandler {
	return &OperatorHandler{cfg: cfg, auditRepo: auditRepo}
}

// LoginPage renders the operator password form.  If the request already
// carries a valid operator cookie, the user is redirected to the dashboard.
func (h *OperatorHandler) LoginPage(c *gin.Context) {
	if isOperator(c, h.cfg.OperatorSessionSecret) {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "auth/operator_login.html", gin.H{
		"CSRFToken":         CSRFToken(c),
		"Title":             "Operator Login",
		"OperatorLoginPath": h.cfg.OperatorLoginPath,
	})
}

// Login validates the operator password.
func (h *OperatorHandler) Login(c *gin.Context) {
	password := c.PostForm("password")

	valid := false
	if h.cfg.OperatorPasswordHash != "" {
		valid = utils.CheckPassword(password, h.cfg.OperatorPasswordHash)
	} else if h.cfg.OperatorPassword != "" {
		valid = password == h.cfg.OperatorPassword
	}

	if !valid {
		_ = h.auditRepo.Log(c.Request.Context(), &models.AuditLog{
			Action:     "operator_login_failed",
			EntityType: "operator",
			Payload: map[string]interface{}{
				"ip": c.ClientIP(),
			},
		})
		c.HTML(http.StatusUnauthorized, "auth/operator_login.html", gin.H{
			"CSRFToken":         CSRFToken(c),
			"Title":             "Operator Login",
			"Error":             "Invalid password",
			"OperatorLoginPath": h.cfg.OperatorLoginPath,
		})
		return
	}

	_ = h.auditRepo.Log(c.Request.Context(), &models.AuditLog{
		Action:     "operator_login_success",
		EntityType: "operator",
		Payload: map[string]interface{}{
			"ip": c.ClientIP(),
		},
	})

	token := utils.SetOperatorCookie(h.cfg.OperatorSessionSecret)
	c.SetSameSite(h.cfg.SessionCookieSameSite)
	c.SetCookie("operator_auth", token, 86400, "/", "", h.cfg.SessionCookieSecure, true)
	c.Redirect(http.StatusFound, "/")
}

// Logout clears the operator authentication cookie.
func (h *OperatorHandler) Logout(c *gin.Context) {
	_ = h.auditRepo.Log(c.Request.Context(), &models.AuditLog{
		Action:     "operator_logout",
		EntityType: "operator",
		Payload: map[string]interface{}{
			"ip": c.ClientIP(),
		},
	})
	c.SetSameSite(h.cfg.SessionCookieSameSite)
	c.SetCookie("operator_auth", "", -1, "/", "", h.cfg.SessionCookieSecure, true)
	c.Redirect(http.StatusFound, h.cfg.OperatorLoginPath)
}

// isOperator reports whether the request carries a valid operator cookie.
func isOperator(c *gin.Context, secret string) bool {
	token, err := c.Cookie("operator_auth")
	if err != nil {
		return false
	}
	return utils.VerifyOperatorToken(secret, token)
}
