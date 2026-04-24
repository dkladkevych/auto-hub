// Package service holds the business logic that sits between HTTP handlers
// and the repository layer.  Each service is responsible for one bounded
// context (auth, users, mailboxes, domains, webmail).
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"auto-hub/mail/internal/config"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
	"auto-hub/mail/internal/utils"
)

// AuthService manages user authentication, session creation and validation.
type AuthService struct {
	userRepo    *repo.UserRepo
	sessionRepo *repo.SessionRepo
	cfg         *config.Config
}

// NewAuthService creates an AuthService with the required repositories and
// application configuration.
func NewAuthService(userRepo *repo.UserRepo, sessionRepo *repo.SessionRepo, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
	}
}

// generateSessionToken creates a cryptographically secure random token.
func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 digest of a raw token.  Only the digest is
// ever stored in the database so that a DB leak does not expose valid
// session tokens.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// Login verifies the email/password pair and, on success, creates a new
// session and returns the raw token (which must be written to a cookie).
func (s *AuthService) Login(ctx context.Context, email, password, userAgent, ip string) (*models.Session, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if user == nil || !user.IsActive {
		return nil, "", fmt.Errorf("invalid credentials")
	}
	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	token, err := generateSessionToken()
	if err != nil {
		return nil, "", err
	}

	session := &models.Session{
		UserID:           user.ID,
		SessionTokenHash: hashToken(token),
		UserAgent:        userAgent,
		IPAddress:        ip,
		ExpiresAt:        time.Now().Add(s.cfg.SessionMaxAge),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, "", err
	}

	return session, token, nil
}

// Logout destroys a session by its primary key.
func (s *AuthService) Logout(ctx context.Context, sessionID int) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

// ValidateSession checks whether a raw token corresponds to a non-expired
// session and returns the session together with the associated user.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*models.Session, *models.User, error) {
	if token == "" {
		return nil, nil, fmt.Errorf("no token")
	}
	session, err := s.sessionRepo.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, fmt.Errorf("session not found or expired")
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || !user.IsActive {
		return nil, nil, fmt.Errorf("user not found or inactive")
	}

	_ = s.sessionRepo.UpdateLastSeen(ctx, session.ID)
	return session, user, nil
}

// CleanExpiredSessions removes every session whose expires_at is in the
// past.  Call this periodically to keep the sessions table small.
func (s *AuthService) CleanExpiredSessions(ctx context.Context) error {
	return s.sessionRepo.DeleteExpired(ctx)
}
