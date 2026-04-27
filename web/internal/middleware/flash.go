package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const flashSessionName = "flash_session"

// FlashMiddleware extracts and clears flash messages.
func FlashMiddleware(store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, flashSessionName)
		flashes := session.Flashes()
		if len(flashes) > 0 {
			session.Save(c.Request, c.Writer)
			c.Set("flashes", flashes)
		}
		c.Next()
	}
}

// AddFlash adds a flash message to the session.
func AddFlash(c *gin.Context, store *sessions.CookieStore, msg string) {
	session, _ := store.Get(c.Request, flashSessionName)
	session.AddFlash(msg)
	session.Save(c.Request, c.Writer)
}
