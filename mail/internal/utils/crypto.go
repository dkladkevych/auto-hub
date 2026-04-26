// Package utils holds small, reusable helpers that do not belong to any
// specific layer of the application.
package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates a bcrypt hash from a plain-text password.  The
// cost parameter is left at bcrypt.DefaultCost which is suitable for
// interactive login use.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a plain-text password against a bcrypt hash.
// It returns true only when the hash was produced from the exact same
// password.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SignToken creates an HMAC-SHA256 signature for a token using the given secret.
func SignToken(token, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyToken checks whether the signature matches the token and secret.
func VerifyToken(token, signature, secret string) bool {
	expected := SignToken(token, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// VerifySignedToken splits a signed token (token.signature) and verifies it.
// Returns the raw token and true on success.
func VerifySignedToken(signed, secret string) (string, bool) {
	parts := strings.Split(signed, ".")
	if len(parts) != 2 {
		return "", false
	}
	if VerifyToken(parts[0], parts[1], secret) {
		return parts[0], true
	}
	return "", false
}
