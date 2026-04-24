// Package utils holds small, reusable helpers that do not belong to any
// specific layer of the application.
package utils

import "golang.org/x/crypto/bcrypt"

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
