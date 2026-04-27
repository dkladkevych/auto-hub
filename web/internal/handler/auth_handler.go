package handler

import (
	"net/http"
	"strings"

	"auto-hub/web/internal/middleware"
	"auto-hub/web/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// AuthHandler handles user authentication routes.
type AuthHandler struct {
	authService  *service.AuthService
	emailService *service.EmailService
	store        *sessions.CookieStore
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService, emailService *service.EmailService, store *sessions.CookieStore) *AuthHandler {
	return &AuthHandler{authService: authService, emailService: emailService, store: store}
}

// RegisterPage handles GET /register.
func (h *AuthHandler) RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth_register", BaseData(c))
}

// Register handles POST /register.
func (h *AuthHandler) Register(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	fullName := strings.TrimSpace(c.PostForm("full_name"))
	password := c.PostForm("password")
	confirm := c.PostForm("confirm_password")

	data := BaseData(c)
	if email == "" || password == "" {
		data["Error"] = "Email and password are required"
		c.HTML(http.StatusBadRequest, "auth_register", data)
		return
	}
	if password != confirm {
		data["Error"] = "Passwords do not match"
		c.HTML(http.StatusBadRequest, "auth_register", data)
		return
	}

	_, code, err := h.authService.RegisterUser(email, password, fullName)
	if err != nil {
		data["Error"] = err.Error()
		c.HTML(http.StatusBadRequest, "auth_register", data)
		return
	}

	_ = h.emailService.SendVerificationEmail(email, code)
	c.Redirect(http.StatusFound, "/verify?email="+email)
}

// VerifyPage handles GET /verify.
func (h *AuthHandler) VerifyPage(c *gin.Context) {
	data := BaseData(c)
	data["Email"] = strings.TrimSpace(strings.ToLower(c.Query("email")))
	c.HTML(http.StatusOK, "auth_verify", data)
}

// Verify handles POST /verify.
func (h *AuthHandler) Verify(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	code := strings.TrimSpace(c.PostForm("code"))

	if ok, _ := h.authService.VerifyUser(email, code); ok {
		middleware.AddFlash(c, h.store, "Email verified! You can now log in.")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	data := BaseData(c)
	data["Email"] = email
	data["Error"] = "Invalid or expired verification code"
	c.HTML(http.StatusOK, "auth_verify", data)
}

// LoginPage handles GET /login.
func (h *AuthHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth_login", BaseData(c))
}

// Login handles POST /login.
func (h *AuthHandler) Login(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")

	user, err := h.authService.AuthenticateUser(email, password)
	if err != nil {
		data := BaseData(c)
		data["Error"] = "Invalid email or password, or account not verified"
		c.HTML(http.StatusOK, "auth_login", data)
		return
	}

	session, _ := h.store.Get(c.Request, userSessionName)
	session.Values["user_id"] = user.ID
	session.Save(c.Request, c.Writer)
	c.Redirect(http.StatusFound, "/profile")
}

// Logout handles POST /logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	session, _ := h.store.Get(c.Request, userSessionName)
	delete(session.Values, "user_id")
	session.Save(c.Request, c.Writer)
	c.Redirect(http.StatusFound, "/")
}

// Profile handles GET /profile.
func (h *AuthHandler) Profile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.authService.GetUserByID(userID.(int))
	if err != nil {
		session, _ := h.store.Get(c.Request, userSessionName)
		delete(session.Values, "user_id")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/login")
		return
	}
	data := BaseData(c)
	data["User"] = user
	c.HTML(http.StatusOK, "auth_profile", data)
}

const userSessionName = "user_session"
