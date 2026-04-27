package service

import (
	"context"
	"fmt"

	"auto-hub/mail/internal/mailprovider"
	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
)

// InternalAPIService handles server-to-server email requests from the
// Auto-Hub web service (e.g. verification codes).
type InternalAPIService struct {
	mailboxRepo  *repo.MailboxRepo
	mailProvider mailprovider.MailProvider
	auditRepo    *repo.AuditRepo
}

// NewInternalAPIService wires the service.
func NewInternalAPIService(mailboxRepo *repo.MailboxRepo, mailProvider mailprovider.MailProvider, auditRepo *repo.AuditRepo) *InternalAPIService {
	return &InternalAPIService{
		mailboxRepo:  mailboxRepo,
		mailProvider: mailProvider,
		auditRepo:    auditRepo,
	}
}

// Send validates that `from` is an active system mailbox with CanSend=true,
// dispatches the message through the configured MailProvider, and writes an
// audit log.
func (s *InternalAPIService) Send(ctx context.Context, from, to, subject, text, html string) error {
	m, err := s.mailboxRepo.GetByEmail(ctx, from)
	if err != nil {
		return fmt.Errorf("lookup mailbox: %w", err)
	}
	if m == nil {
		return fmt.Errorf("mailbox not found")
	}
	if !m.IsActive {
		return fmt.Errorf("mailbox is inactive")
	}
	if !m.CanSend {
		return fmt.Errorf("mailbox cannot send")
	}
	if m.MailboxType != "system" {
		return fmt.Errorf("mailbox is not a system mailbox")
	}

	msg := &models.OutgoingMessage{
		To:       to,
		Subject:  subject,
		TextBody: text,
		HTMLBody: html,
	}

	if err := s.mailProvider.SendMessage(ctx, from, msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	log := buildAuditLog(0, "internal_api_send", "message", nil, map[string]interface{}{
		"from":    from,
		"to":      to,
		"subject": subject,
	})
	_ = s.auditRepo.Log(ctx, log)

	return nil
}
