package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

const csrfTokenKey = "csrf_token"

// generateToken creates a random hex token.
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CSRFMiddleware generates and validates CSRF tokens.
func CSRFMiddleware(store *sessions.CookieStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := store.Get(c.Request, "csrf_session")
		token, ok := session.Values[csrfTokenKey].(string)
		if !ok || token == "" {
			token = generateToken()
			session.Values[csrfTokenKey] = token
			session.Save(c.Request, c.Writer)
		}
		c.Set("csrf_token", token)

		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
			formToken := c.PostForm("csrf_token")
			if formToken == "" {
				formToken = c.GetHeader("X-CSRF-Token")
			}
			if subtle.ConstantTimeCompare([]byte(token), []byte(formToken)) != 1 {
				c.String(http.StatusForbidden, "Invalid CSRF token")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
