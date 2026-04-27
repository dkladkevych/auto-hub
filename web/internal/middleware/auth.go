package middleware

import (
	"net/http"

	"auto-hub/web/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const (
	adminSessionName = "admin_session"
	userSessionName  = "user_session"
)

// AdminRequired ensures the request is authenticated as admin.
func AdminRequired(cfg *config.Config, store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, adminSessionName)
		auth, ok := session.Values["authenticated"].(bool)
		if !ok || !auth {
			c.Redirect(http.StatusFound, "/"+cfg.AdminPath+"/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// LoginRequired ensures the request is authenticated as a user.
func LoginRequired(store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, userSessionName)
		userID, ok := session.Values["user_id"].(int)
		if !ok || userID == 0 {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

// InjectUser loads the current user into context if available.
func InjectUser(store *sessions.CookieStore, userGetter func(int) interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, userSessionName)
		userID, ok := session.Values["user_id"].(int)
		if ok && userID > 0 {
			c.Set("current_user", userGetter(userID))
		}
		c.Next()
	}
}

// AdminPathMiddleware injects admin path into context.
func AdminPathMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("admin_path", cfg.AdminPath)
		c.Next()
	}
}
