// Package utils (operator.go) provides HMAC-based token creation and
// verification for the operator (super-admin) login cookie.  The token
// embeds a timestamp so that it naturally expires after 24 hours.
package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SignOperatorToken builds a signed token that encodes the current time.
// The signature is an HMAC-SHA256 over the string "operator:<timestamp>".
func SignOperatorToken(secret string, timestamp int64) string {
	msg := fmt.Sprintf("operator:%d", timestamp)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("operator:%d:%s", timestamp, sig)))
}

// VerifyOperatorToken decodes a token, checks its age (max 24 hours) and
// verifies the HMAC signature against the shared secret.
func VerifyOperatorToken(secret, token string) bool {
	b, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	parts := strings.SplitN(string(b), ":", 3)
	if len(parts) != 3 || parts[0] != "operator" {
		return false
	}
	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	// max age 24 hours
	if time.Now().Unix()-ts > 86400 {
		return false
	}
	expected := SignOperatorToken(secret, ts)
	return hmac.Equal([]byte(token), []byte(expected))
}

// SetOperatorCookie generates a fresh token for the operator cookie.
func SetOperatorCookie(secret string) string {
	ts := time.Now().Unix()
	return SignOperatorToken(secret, ts)
}
