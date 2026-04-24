package handler

import (
	"net/http"

	"auto-hub/mail/internal/config"
	"auto-hub/mail/internal/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// OperatorHandler manages the super-admin login flow.  The operator is a
// special account that exists outside of the users table and is
// authenticated via a standalone cookie with an HMAC token.
type OperatorHandler struct {
	cfg *config.Config
}

// NewOperatorHandler creates an OperatorHandler.
func NewOperatorHandler(cfg *config.Config) *OperatorHandler {
	return &OperatorHandler{cfg: cfg}
}

// LoginPage renders the operator password form.  If the request already
// carries a valid operator cookie, the user is redirected to the dashboard.
func (h *OperatorHandler) LoginPage(c *gin.Context) {
	if isOperator(c, h.cfg.OperatorSessionSecret) {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "operator_login.html", gin.H{
		"Title": "Operator Login",
	})
}

// Login validates the operator password.  In production the password must
// be provided as a bcrypt hash (OPERATOR_PASSWORD_HASH); a plain-text
// fallback (OPERATOR_PASSWORD) is supported for local development only.
func (h *OperatorHandler) Login(c *gin.Context) {
	password := c.PostForm("password")

	valid := false
	if h.cfg.OperatorPasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(h.cfg.OperatorPasswordHash), []byte(password)); err == nil {
			valid = true
		}
	} else if h.cfg.OperatorPassword != "" {
		valid = password == h.cfg.OperatorPassword
	}

	if !valid {
		c.HTML(http.StatusUnauthorized, "operator_login.html", gin.H{
			"Title": "Operator Login",
			"Error": "Invalid password",
		})
		return
	}

	token := utils.SetOperatorCookie(h.cfg.OperatorSessionSecret)
	c.SetCookie("operator_auth", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// Logout clears the operator authentication cookie.
func (h *OperatorHandler) Logout(c *gin.Context) {
	c.SetCookie("operator_auth", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/operator/login")
}

// isOperator reports whether the request carries a valid operator cookie.
func isOperator(c *gin.Context, secret string) bool {
	token, err := c.Cookie("operator_auth")
	if err != nil {
		return false
	}
	return utils.VerifyOperatorToken(secret, token)
}
