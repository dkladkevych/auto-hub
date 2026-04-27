package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"auto-hub/web/internal/models"
	"auto-hub/web/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user authentication.
type AuthService struct {
	userRepo *repo.UserRepo
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repo.UserRepo) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// RegisterUser creates a new unverified user.
func (s *AuthService) RegisterUser(email, password, fullName string) (int, string, error) {
	if len(password) < 6 {
		return 0, "", fmt.Errorf("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, "", err
	}
	code, err := generateCode()
	if err != nil {
		return 0, "", err
	}
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	id, err := s.userRepo.Create(email, string(hash), fullName, code, expiresAt)
	if err != nil {
		return 0, "", fmt.Errorf("a user with this email already exists")
	}
	return id, code, nil
}

// VerifyUser confirms email verification code.
func (s *AuthService) VerifyUser(email, code string) (bool, error) {
	return s.userRepo.Verify(email, code)
}

// AuthenticateUser checks credentials and returns user if valid and verified.
func (s *AuthService) AuthenticateUser(email, password string) (*models.User, error) {
	user, err := s.userRepo.GetVerifiedByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}
	return user, nil
}

// GetUserByID fetches a user by ID.
func (s *AuthService) GetUserByID(id int) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}
