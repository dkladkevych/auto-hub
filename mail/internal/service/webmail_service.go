package service

import (
	"context"
	"fmt"

	"auto-hub/mail/internal/mailprovider"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
)

// WebmailService sits between HTTP handlers and the MailProvider abstraction.
// It resolves mailbox IDs to email addresses and enforces membership checks
// so that handlers remain free of business rules.
type WebmailService struct {
	provider    mailprovider.MailProvider
	mailboxRepo *repo.MailboxRepo
	memberRepo  *repo.MailboxMemberRepo
}

// NewWebmailService wires the service to a concrete MailProvider implementation
// (currently the in-memory mock; later this can be swapped for IMAP/SMTP).
func NewWebmailService(provider mailprovider.MailProvider, mailboxRepo *repo.MailboxRepo, memberRepo *repo.MailboxMemberRepo) *WebmailService {
	return &WebmailService{
		provider:    provider,
		mailboxRepo: mailboxRepo,
		memberRepo:  memberRepo,
	}
}

// CanAccess returns true if the user is allowed to interact with the given
// mailbox.  Operators have full access; everyone else must either own the
// mailbox or be a member of it.
func (s *WebmailService) CanAccess(ctx context.Context, user *models.User, mailboxID int) (bool, error) {
	if user.Role == "operator" {
		return true, nil
	}
	m, err := s.mailboxRepo.GetByID(ctx, mailboxID)
	if err != nil {
		return false, err
	}
	if m == nil {
		return false, nil
	}
	if !m.IsActive {
		return false, nil
	}
	if m.Email == user.Email {
		return true, nil
	}
	exists, err := s.memberRepo.Exists(ctx, user.ID, mailboxID)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// requireMailbox fetches the mailbox and checks it is active.
func (s *WebmailService) requireMailbox(ctx context.Context, mailboxID int) (*models.Mailbox, error) {
	m, err := s.mailboxRepo.GetByID(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("mailbox not found")
	}
	if !m.IsActive {
		return nil, fmt.Errorf("mailbox is inactive")
	}
	return m, nil
}

// GetMailboxByID returns a mailbox by its database ID.
func (s *WebmailService) GetMailboxByID(ctx context.Context, mailboxID int) (*models.Mailbox, error) {
	m, err := s.mailboxRepo.GetByID(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("mailbox not found")
	}
	return m, nil
}

// ListFolders returns the folder list (Inbox, Sent, Drafts, Trash, etc.)
// for a mailbox.
func (s *WebmailService) ListFolders(ctx context.Context, mailboxID int) ([]models.Folder, error) {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	return s.provider.ListFolders(ctx, m.Email)
}

// ListMessages returns a paginated slice of messages inside a folder.
func (s *WebmailService) ListMessages(ctx context.Context, mailboxID int, folder string, limit, offset int) ([]models.Message, error) {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	return s.provider.ListMessages(ctx, m.Email, folder, limit, offset)
}

// CountMessages returns the total number of messages in a folder.
func (s *WebmailService) CountMessages(ctx context.Context, mailboxID int, folder string) (int, error) {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return 0, err
	}
	return s.provider.CountMessages(ctx, m.Email, folder)
}

// GetMessage retrieves a single message by its provider-specific ID.
func (s *WebmailService) GetMessage(ctx context.Context, mailboxID int, folder, messageID string) (*models.Message, error) {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return nil, err
	}
	return s.provider.GetMessage(ctx, m.Email, folder, messageID)
}

// SendMessage dispatches an outgoing message through the provider.
func (s *WebmailService) SendMessage(ctx context.Context, mailboxID int, msg *models.OutgoingMessage) error {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return err
	}
	return s.provider.SendMessage(ctx, m.Email, msg)
}

// MarkSeen toggles the \\Seen flag for a single message.
func (s *WebmailService) MarkSeen(ctx context.Context, mailboxID int, folder, messageID string, seen bool) error {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return err
	}
	return s.provider.MarkSeen(ctx, m.Email, folder, messageID, seen)
}

// SetFlagged updates the starred state of a message.
func (s *WebmailService) SetFlagged(ctx context.Context, mailboxID int, folder, messageID string, flagged bool) error {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return err
	}
	return s.provider.SetFlagged(ctx, m.Email, folder, messageID, flagged)
}

// DeleteMessage moves a message to the Trash folder.
func (s *WebmailService) DeleteMessage(ctx context.Context, mailboxID int, folder, messageID string) error {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return err
	}
	return s.provider.DeleteMessage(ctx, m.Email, folder, messageID)
}

// SaveDraft persists a draft message.
func (s *WebmailService) SaveDraft(ctx context.Context, mailboxID int, msg *models.OutgoingMessage) error {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return err
	}
	return s.provider.SaveDraft(ctx, m.Email, msg)
}

// EmptyTrash permanently removes every message currently in the Trash folder.
func (s *WebmailService) EmptyTrash(ctx context.Context, mailboxID int) error {
	m, err := s.requireMailbox(ctx, mailboxID)
	if err != nil {
		return err
	}
	return s.provider.EmptyTrash(ctx, m.Email)
}

// ListAccessibleMailboxes returns all mailboxes a user can view in the
// webmail sidebar.  Operators see everything; regular users see their own
// personal mailbox plus any shared mailbox they are a member of.
func (s *WebmailService) ListAccessibleMailboxes(ctx context.Context, user *models.User) ([]models.Mailbox, error) {
	all, err := s.mailboxRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	if user.Role == "operator" {
		var out []models.Mailbox
		for _, m := range all {
			if m.IsActive {
				out = append(out, m)
			}
		}
		return out, nil
	}

	var out []models.Mailbox
	for _, m := range all {
		if !m.IsActive {
			continue
		}
		if m.Email == user.Email {
			out = append(out, m)
			continue
		}
		exists, err := s.memberRepo.Exists(ctx, user.ID, m.ID)
		if err != nil {
			continue
		}
		if exists {
			out = append(out, m)
		}
	}
	return out, nil
}

// ListSendableMailboxes returns the subset of accessible mailboxes from
// which the user is allowed to send messages.  Read-only members are
// excluded.
func (s *WebmailService) ListSendableMailboxes(ctx context.Context, user *models.User) ([]models.Mailbox, error) {
	accessible, err := s.ListAccessibleMailboxes(ctx, user)
	if err != nil {
		return nil, err
	}
	var out []models.Mailbox
	for _, m := range accessible {
		if !m.IsActive || !m.CanSend {
			continue
		}
		if user.Role == "operator" || m.Email == user.Email {
			out = append(out, m)
			continue
		}
		// For shared mailboxes, check the user is not read_only
		members, err := s.memberRepo.ListByMailbox(ctx, m.ID)
		if err != nil {
			continue
		}
		for _, mem := range members {
			if mem.UserID == user.ID && mem.AccessRole != "read_only" {
				out = append(out, m)
				break
			}
		}
	}
	return out, nil
}
