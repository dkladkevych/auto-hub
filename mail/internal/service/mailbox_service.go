package service

import (
	"context"
	"fmt"
	"strings"

	"auto-hub/mail/internal/maildir"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
	"auto-hub/mail/internal/utils"
)

// MailboxService manages mailboxes (shared, system and personal) together
// with their memberships and settings.
type MailboxService struct {
	mailboxRepo *repo.MailboxRepo
	memberRepo  *repo.MailboxMemberRepo
	auditRepo   *repo.AuditRepo
}

// NewMailboxService creates a MailboxService with the required repositories.
func NewMailboxService(mailboxRepo *repo.MailboxRepo, memberRepo *repo.MailboxMemberRepo, auditRepo *repo.AuditRepo) *MailboxService {
	return &MailboxService{
		mailboxRepo: mailboxRepo,
		memberRepo:  memberRepo,
		auditRepo:   auditRepo,
	}
}

// buildMaildirPath returns the canonical filesystem path for a mailbox's
// Maildir.  This path is stored in the database and used by the real
// Postfix/Dovecot backend in production.
func buildMaildirPath(domain, localPart string) string {
	return fmt.Sprintf("/var/mail/vhosts/%s/%s/", domain, localPart)
}

// Create registers a new shared or system mailbox.  Personal mailboxes are
// created automatically by UserService.Create and cannot be created here.
func (s *MailboxService) Create(ctx context.Context, actorID int, localPart, domain, displayName, mailboxType string, canReceive, canSend bool, quotaMb int, password string) (*models.Mailbox, error) {
	localPart = strings.TrimSpace(localPart)
	domain = strings.TrimSpace(domain)
	if localPart == "" || domain == "" {
		return nil, fmt.Errorf("local part and domain are required")
	}
	if mailboxType == "personal" {
		return nil, fmt.Errorf("personal mailboxes can only be created alongside users")
	}

	email := localPart + "@" + domain

	var passwordHash string
	if password != "" {
		hash, err := utils.HashPassword(password)
		if err != nil {
			return nil, err
		}
		passwordHash = hash
	}

	m := &models.Mailbox{
		Email:               email,
		LocalPart:           localPart,
		Domain:              domain,
		DisplayName:         displayName,
		MailboxType:         mailboxType,
		IsActive:            true,
		CanReceive:          canReceive,
		CanSend:             canSend,
		QuotaMb:             quotaMb,
		MaildirPath:         buildMaildirPath(domain, localPart),
		ImapLoginEnabled:    true,
		SmtpLoginEnabled:    true,
		MailboxPasswordHash: passwordHash,
		CreatedBy:           &actorID,
	}

	if err := s.mailboxRepo.Create(ctx, m); err != nil {
		return nil, err
	}

	if err := maildir.Create(m.MaildirPath); err != nil {
		// Rollback DB record on filesystem error
		_ = s.mailboxRepo.Delete(ctx, m.ID)
		return nil, fmt.Errorf("maildir creation failed: %w", err)
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "mailbox_created", "mailbox", &m.ID, map[string]interface{}{
		"email":        email,
		"type":         mailboxType,
		"maildir_path": m.MaildirPath,
		"quota_mb":     quotaMb,
		"has_password": passwordHash != "",
	}))

	return m, nil
}

// List returns every mailbox in the database.
func (s *MailboxService) List(ctx context.Context) ([]models.Mailbox, error) {
	return s.mailboxRepo.List(ctx)
}

// ListPaginated returns a paginated slice of mailboxes together with the
// total count so the caller can render page navigation.
func (s *MailboxService) ListPaginated(ctx context.Context, page, perPage int) ([]models.Mailbox, int, error) {
	return s.ListFilteredPaginated(ctx, "", nil, "", page, perPage)
}

// ListFilteredPaginated returns mailboxes filtered by type, active state and
// search term with pagination metadata.
func (s *MailboxService) ListFilteredPaginated(ctx context.Context, mailboxType string, active *bool, search string, page, perPage int) ([]models.Mailbox, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	mailboxes, err := s.mailboxRepo.ListFilteredPaginated(ctx, mailboxType, active, search, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.mailboxRepo.CountFiltered(ctx, mailboxType, active, search)
	if err != nil {
		return nil, 0, err
	}
	return mailboxes, total, nil
}

// GetByID fetches a single mailbox by primary key.
func (s *MailboxService) GetByID(ctx context.Context, id int) (*models.Mailbox, error) {
	return s.mailboxRepo.GetByID(ctx, id)
}

// GetWithMembers returns a mailbox together with its current memberships.
func (s *MailboxService) GetWithMembers(ctx context.Context, id int) (*models.Mailbox, []models.MailboxMember, error) {
	m, err := s.mailboxRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if m == nil {
		return nil, nil, fmt.Errorf("mailbox not found")
	}
	members, err := s.memberRepo.ListByMailbox(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return m, members, nil
}

// Update modifies a mailbox's display name, type and quota.  Personal
// mailboxes are rejected because they are managed via the user page.
func (s *MailboxService) Update(ctx context.Context, actorID, mailboxID int, displayName, mailboxType string, quotaMb int) error {
	m, err := s.mailboxRepo.GetByID(ctx, mailboxID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("mailbox not found")
	}
	if m.MailboxType == "personal" {
		return fmt.Errorf("personal mailbox settings must be edited via the user page")
	}
	if mailboxType == "personal" {
		return fmt.Errorf("cannot change mailbox type to personal")
	}

	m.DisplayName = displayName
	m.MailboxType = mailboxType
	m.QuotaMb = quotaMb

	if err := s.mailboxRepo.Update(ctx, m); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "mailbox_updated", "mailbox", &mailboxID, map[string]interface{}{
		"display_name": displayName,
		"type":         mailboxType,
		"quota_mb":     quotaMb,
	}))

	return nil
}

// Delete soft-deletes a mailbox and renames its Maildir.
func (s *MailboxService) Delete(ctx context.Context, actorID, mailboxID int) error {
	m, err := s.mailboxRepo.GetByID(ctx, mailboxID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("mailbox not found")
	}

	if err := s.mailboxRepo.Delete(ctx, mailboxID); err != nil {
		return err
	}

	if _, err := maildir.SoftDelete(m.MaildirPath); err != nil {
		return fmt.Errorf("maildir soft-delete failed: %w", err)
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "mailbox_deleted", "mailbox", &mailboxID, map[string]interface{}{"email": m.Email}))

	return nil
}

// Reactivate restores a soft-deleted mailbox and its Maildir.
func (s *MailboxService) Reactivate(ctx context.Context, actorID, mailboxID int) error {
	m, err := s.mailboxRepo.GetByID(ctx, mailboxID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("mailbox not found")
	}
	if m.IsActive {
		return fmt.Errorf("mailbox is already active")
	}

	// Try to restore Maildir if a deleted copy exists
	if deletedPath, err := maildir.FindDeleted(m.MaildirPath); err == nil {
		if err := maildir.Restore(deletedPath, m.MaildirPath); err != nil {
			return fmt.Errorf("maildir restore failed: %w", err)
		}
	}

	if err := s.mailboxRepo.Reactivate(ctx, mailboxID); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "mailbox_reactivated", "mailbox", &mailboxID, map[string]interface{}{"email": m.Email}))

	return nil
}

// UpdateSettings toggles the can_receive and can_send flags for a mailbox.
func (s *MailboxService) UpdateSettings(ctx context.Context, actorID, mailboxID int, canReceive, canSend bool) error {
	m, err := s.mailboxRepo.GetByID(ctx, mailboxID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("mailbox not found")
	}

	if err := s.mailboxRepo.UpdateSettings(ctx, mailboxID, canReceive, canSend); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "mailbox_settings_updated", "mailbox", &mailboxID, map[string]interface{}{
		"previous_can_receive": m.CanReceive,
		"previous_can_send":    m.CanSend,
		"new_can_receive":      canReceive,
		"new_can_send":         canSend,
	}))

	return nil
}

// SetPassword hashes and stores a new password for a mailbox.
func (s *MailboxService) SetPassword(ctx context.Context, actorID, mailboxID int, password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	if err := s.mailboxRepo.UpdatePassword(ctx, mailboxID, hash); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "mailbox_password_set", "mailbox", &mailboxID, map[string]interface{}{}))

	return nil
}

// ResetPassword clears a mailbox's password hash so that the account
// cannot be accessed until a new password is configured.
func (s *MailboxService) ResetPassword(ctx context.Context, actorID, mailboxID int) error {
	if err := s.mailboxRepo.UpdatePassword(ctx, mailboxID, ""); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "mailbox_password_reset", "mailbox", &mailboxID, map[string]interface{}{}))

	return nil
}

// AddMember links a user to a mailbox with the given access role.
func (s *MailboxService) AddMember(ctx context.Context, actorID, mailboxID, userID int, accessRole string) (*models.MailboxMember, error) {
	exists, err := s.memberRepo.Exists(ctx, userID, mailboxID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("user is already a member of this mailbox")
	}

	member := &models.MailboxMember{
		UserID:     userID,
		MailboxID:  mailboxID,
		AccessRole: accessRole,
	}

	if err := s.memberRepo.Add(ctx, member); err != nil {
		return nil, err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "member_added", "mailbox_member", &member.ID, map[string]interface{}{
		"mailbox_id":  mailboxID,
		"user_id":     userID,
		"access_role": accessRole,
	}))

	return member, nil
}

// RemoveMember deletes a membership by its primary key.
func (s *MailboxService) RemoveMember(ctx context.Context, actorID, memberID int) error {
	member, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return err
	}
	if member == nil {
		return fmt.Errorf("member not found")
	}

	if err := s.memberRepo.Remove(ctx, memberID); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "member_removed", "mailbox_member", &memberID, map[string]interface{}{
		"mailbox_id": member.MailboxID,
		"user_id":    member.UserID,
	}))

	return nil
}
