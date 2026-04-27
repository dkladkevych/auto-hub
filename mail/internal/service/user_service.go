package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"auto-hub/mail/internal/maildir"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
	"auto-hub/mail/internal/utils"
)

// UserService handles creation, modification and deletion of user accounts.
// Creating a user always creates a matching personal mailbox with the same
// email address.
type UserService struct {
	userRepo    *repo.UserRepo
	mailboxRepo *repo.MailboxRepo
	memberRepo  *repo.MailboxMemberRepo
	auditRepo   *repo.AuditRepo
}

// NewUserService creates a UserService with the required repositories.
func NewUserService(userRepo *repo.UserRepo, mailboxRepo *repo.MailboxRepo, memberRepo *repo.MailboxMemberRepo, auditRepo *repo.AuditRepo) *UserService {
	return &UserService{
		userRepo:    userRepo,
		mailboxRepo: mailboxRepo,
		memberRepo:  memberRepo,
		auditRepo:   auditRepo,
	}
}

// canCreateRole enforces the rule that operators may create admins and
// regular users, whereas admins may only create regular users.
func (s *UserService) canCreateRole(actorRole, targetRole string) error {
	switch actorRole {
	case "operator":
		if targetRole != "admin" && targetRole != "user" {
			return fmt.Errorf("invalid role")
		}
		return nil
	case "admin":
		if targetRole != "user" {
			return fmt.Errorf("admin can only create users")
		}
		return nil
	default:
		return fmt.Errorf("insufficient permissions to create users")
	}
}

// Create registers a new user, hashes their password, creates a personal
// mailbox, and adds a manager membership so the user owns their own mailbox.
func (s *UserService) Create(ctx context.Context, actorID int, username, password, fullName, role, domain string, quotaMb int, canReceive, canSend bool) (*models.User, *models.Mailbox, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, nil, fmt.Errorf("username is required")
	}

	email := username + "@" + domain

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, fmt.Errorf("user with this email already exists")
	}

	actorRole := "operator"
	if actorID != 0 {
		actor, err := s.userRepo.GetByID(ctx, actorID)
		if err != nil {
			return nil, nil, err
		}
		if actor == nil {
			return nil, nil, fmt.Errorf("actor not found")
		}
		actorRole = actor.Role
	}

	if err := s.canCreateRole(actorRole, role); err != nil {
		return nil, nil, err
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, nil, err
	}

	createdBy := actorID
	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		FullName:     fullName,
		Role:         role,
		IsActive:     true,
		CreatedBy:    &createdBy,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	maildirPath := fmt.Sprintf("/var/mail/vhosts/%s/%s/", domain, username)
	mailbox := &models.Mailbox{
		Email:               email,
		LocalPart:           username,
		Domain:              domain,
		DisplayName:         fullName,
		MailboxType:         "personal",
		IsActive:            true,
		CanReceive:          canReceive,
		CanSend:             canSend,
		QuotaMb:             quotaMb,
		MaildirPath:         maildirPath,
		ImapLoginEnabled:    true,
		SmtpLoginEnabled:    true,
		MailboxPasswordHash: "",
		CreatedBy:           &createdBy,
	}

	if err := s.mailboxRepo.Create(ctx, mailbox); err != nil {
		_ = s.userRepo.Delete(ctx, user.ID)
		return nil, nil, fmt.Errorf("failed to create mailbox: %w", err)
	}

	if err := maildir.Create(mailbox.MaildirPath); err != nil {
		log.Printf("maildir create warning for %s: %v", mailbox.MaildirPath, err)
	}

	member := &models.MailboxMember{
		UserID:     user.ID,
		MailboxID:  mailbox.ID,
		AccessRole: "manager",
	}
	_ = s.memberRepo.Add(ctx, member)

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "user_created", "user", &user.ID, map[string]interface{}{
		"email":     email,
		"full_name": fullName,
		"role":      role,
	}))

	return user, mailbox, nil
}

// List returns all users ordered by creation time (newest first).
func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	return s.userRepo.List(ctx)
}

// GetByID fetches a single user by primary key.
func (s *UserService) GetByID(ctx context.Context, id int) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetByEmail fetches a single user by their exact email address.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// GetWithMailbox returns a user together with their associated personal
// mailbox (identified by matching email and type "personal").
func (s *UserService) GetWithMailbox(ctx context.Context, id int) (*models.User, *models.Mailbox, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}
	mailboxes, err := s.mailboxRepo.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, m := range mailboxes {
		if m.Email == user.Email && m.MailboxType == "personal" {
			return user, &m, nil
		}
	}
	return user, nil, nil
}

// Update modifies a user's profile and synchronises the changes to their
// personal mailbox (display name, quota, receive/send flags and password).
func (s *UserService) Update(ctx context.Context, actorID int, id int, fullName, role string, isActive bool, quotaMb int, canReceive, canSend bool, newPassword string) error {
	if actorID == id {
		if !isActive {
			return fmt.Errorf("you cannot deactivate yourself")
		}
		actor, err := s.userRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if actor != nil && actor.Role != role {
			return fmt.Errorf("you cannot change your own role")
		}
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Role change restrictions
	actorRole := "operator"
	if actorID != 0 {
		actor, _ := s.userRepo.GetByID(ctx, actorID)
		if actor != nil {
			actorRole = actor.Role
		}
	}
	if actorRole == "operator" {
		// operator can change any role
	} else if actorRole == "admin" {
		if user.Role != role && role != "user" {
			return fmt.Errorf("admin can only assign user role")
		}
	}

	user.FullName = fullName
	user.Role = role
	user.IsActive = isActive

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Hash new password once if provided
	var newHash string
	if newPassword != "" {
		h, err := utils.HashPassword(newPassword)
		if err != nil {
			return err
		}
		newHash = h
	}

	// Update associated personal mailbox settings
	mailboxes, err := s.mailboxRepo.List(ctx)
	if err == nil {
		for i := range mailboxes {
			m := &mailboxes[i]
			if m.Email == user.Email && m.MailboxType == "personal" {
				m.DisplayName = fullName
				m.QuotaMb = quotaMb
				m.IsActive = isActive
				m.CanReceive = canReceive
				m.CanSend = canSend
				_ = s.mailboxRepo.Update(ctx, m)
				_ = s.mailboxRepo.UpdateSettings(ctx, m.ID, canReceive, canSend)
				break
			}
		}
	}

	// Update user password if provided
	if newHash != "" {
		user.PasswordHash = newHash
		if err := s.userRepo.UpdatePassword(ctx, id, newHash); err != nil {
			return err
		}
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "user_updated", "user", &id, map[string]interface{}{
		"full_name": fullName,
		"role":      role,
		"is_active": isActive,
	}))

	return nil
}

// UpdateProfile allows a regular user to change their own full name and
// password.  Role changes are silently ignored.
func (s *UserService) UpdateProfile(ctx context.Context, userID int, fullName, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	user.FullName = fullName
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	var newHash string
	if newPassword != "" {
		h, err := utils.HashPassword(newPassword)
		if err != nil {
			return err
		}
		newHash = h
		user.PasswordHash = newHash
		if err := s.userRepo.UpdatePassword(ctx, userID, newHash); err != nil {
			return err
		}
	}

	mailboxes, err := s.mailboxRepo.List(ctx)
	if err == nil {
		for i := range mailboxes {
			m := &mailboxes[i]
			if m.Email == user.Email && m.MailboxType == "personal" {
				m.DisplayName = fullName
				_ = s.mailboxRepo.Update(ctx, m)
				break
			}
		}
	}

	return nil
}

// Delete removes a user and also deletes their associated personal mailbox.
// This is safe because mailbox_members has ON DELETE CASCADE.
func (s *UserService) Delete(ctx context.Context, actorID, id int) error {
	if actorID == id {
		return fmt.Errorf("you cannot delete yourself")
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	mailboxes, err := s.mailboxRepo.List(ctx)
	if err == nil {
		for _, m := range mailboxes {
			if m.Email == user.Email && m.MailboxType == "personal" {
				_ = s.mailboxRepo.Delete(ctx, m.ID)
				_, _ = maildir.SoftDelete(m.MaildirPath)
				break
			}
		}
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "user_deleted", "user", &id, map[string]interface{}{"email": user.Email}))

	return nil
}
