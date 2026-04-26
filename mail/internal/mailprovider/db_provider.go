package mailprovider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
)

// DevDBMailProvider is a persistent MailProvider backed by SQLite.
type DevDBMailProvider struct {
	mu          sync.RWMutex
	msgRepo     *repo.MessageRepo
	attRepo     *repo.MessageAttachmentRepo
	mailboxRepo *repo.MailboxRepo
	smtpSender  *SMTPSender
}

// NewDevDBMailProvider creates a DB-backed mail provider.
func NewDevDBMailProvider(msgRepo *repo.MessageRepo, attRepo *repo.MessageAttachmentRepo, mailboxRepo *repo.MailboxRepo, smtpSender *SMTPSender) *DevDBMailProvider {
	return &DevDBMailProvider{msgRepo: msgRepo, attRepo: attRepo, mailboxRepo: mailboxRepo, smtpSender: smtpSender}
}

func (p *DevDBMailProvider) folderList(mailboxEmail string) []models.Folder {
	return []models.Folder{
		{Name: "Inbox", Count: 0, Unseen: 0},
		{Name: "Sent", Count: 0, Unseen: 0},
		{Name: "Drafts", Count: 0, Unseen: 0},
		{Name: "Trash", Count: 0, Unseen: 0},
	}
}

// ListFolders returns all folders for the given mailbox email address.
func (p *DevDBMailProvider) ListFolders(ctx context.Context, mailboxEmail string) ([]models.Folder, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	folders := p.folderList(mailboxEmail)
	out := make([]models.Folder, 0, len(folders)+1)
	for _, f := range folders {
		count, _ := p.msgRepo.CountByFolder(ctx, mailboxEmail, f.Name)
		unseen, _ := p.unseenCount(ctx, mailboxEmail, f.Name)
		f.Count = count
		f.Unseen = unseen
		out = append(out, f)
		if f.Name == "Inbox" {
			starred, _ := p.msgRepo.CountFlagged(ctx, mailboxEmail)
			out = append(out, models.Folder{Name: "Starred", Count: starred, Unseen: 0})
		}
	}
	return out, nil
}

func (p *DevDBMailProvider) unseenCount(ctx context.Context, mailboxEmail, folder string) (int, error) {
	return p.msgRepo.CountUnseen(ctx, mailboxEmail, folder)
}

// ListMessages returns messages in a folder with simple pagination.
func (p *DevDBMailProvider) ListMessages(ctx context.Context, mailboxEmail string, folder string, limit, offset int) ([]models.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if folder == "Trash" {
		_ = p.expireTrash(ctx, mailboxEmail)
	}
	var msgs []models.Message
	var err error
	if folder == "Starred" {
		msgs, err = p.msgRepo.ListFlagged(ctx, mailboxEmail, limit, offset)
	} else {
		msgs, err = p.msgRepo.ListByFolder(ctx, mailboxEmail, folder, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	for i := range msgs {
		p.loadAttachments(ctx, &msgs[i])
	}
	return msgs, nil
}

// CountMessages returns the total number of messages in a folder.
func (p *DevDBMailProvider) CountMessages(ctx context.Context, mailboxEmail string, folder string) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if folder == "Starred" {
		return p.msgRepo.CountFlagged(ctx, mailboxEmail)
	}
	return p.msgRepo.CountByFolder(ctx, mailboxEmail, folder)
}

// GetMessage returns a single message by its ID within a folder.
func (p *DevDBMailProvider) GetMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) (*models.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	msg, err := p.msgRepo.GetByUID(ctx, mailboxEmail, folder, messageID)
	if err != nil {
		return nil, err
	}
	p.loadAttachments(ctx, msg)
	return msg, nil
}

// SendMessage sends an outgoing message. A copy is always placed in the
// sender's Sent folder.  Local recipients receive a copy in their Inbox.
// External recipients are delivered via SMTP when enabled.
func (p *DevDBMailProvider) SendMessage(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	threadID := p.resolveThreadID(ctx, msg.InReplyTo)

	// 1. Sent folder for the sender
	sent := models.Message{
		ID:           generateUID(),
		MailboxEmail: mailboxEmail,
		Folder:       "Sent",
		Subject:      msg.Subject,
		From:         mailboxEmail,
		To:           msg.To,
		Date:         time.Now(),
		TextBody:     msg.TextBody,
		HTMLBody:     msg.HTMLBody,
		Seen:         true,
		InReplyTo:    msg.InReplyTo,
		ThreadID:     threadID,
		Status:       "delivered",
	}
	sentDBID, err := p.msgRepo.Create(ctx, &sent)
	if err != nil {
		return err
	}
	for _, att := range msg.Attachments {
		_, _ = p.attRepo.Create(ctx, sentDBID, att.Filename, att.ContentType, att.Size, att.FilePath)
	}

	// 2. Parse recipients (To + Cc)
	seen := make(map[string]bool)
	var localRecipients []string
	var externalRecipients []string

	for _, field := range []string{msg.To, msg.Cc} {
		for _, raw := range strings.Split(field, ",") {
			addr := strings.TrimSpace(raw)
			if addr == "" || seen[addr] {
				continue
			}
			seen[addr] = true

			mb, err := p.mailboxRepo.GetByEmail(ctx, addr)
			if err == nil && mb != nil && mb.CanReceive {
				localRecipients = append(localRecipients, addr)
			} else {
				externalRecipients = append(externalRecipients, addr)
			}
		}
	}

	// 3. External delivery restrictions
	if len(externalRecipients) > 0 && len(msg.Attachments) > 0 {
		return fmt.Errorf("attachments are not allowed for external recipients yet")
	}

	// 4. Deliver to local recipients
	for _, to := range localRecipients {
		inbox := models.Message{
			ID:           generateUID(),
			MailboxEmail: to,
			Folder:       "Inbox",
			Subject:      msg.Subject,
			From:         mailboxEmail,
			To:           to,
			Date:         time.Now(),
			TextBody:     msg.TextBody,
			HTMLBody:     msg.HTMLBody,
			Seen:         false,
			InReplyTo:    msg.InReplyTo,
			ThreadID:     threadID,
			Status:       "delivered",
		}
		inboxDBID, err := p.msgRepo.Create(ctx, &inbox)
		if err == nil {
			_ = p.attRepo.CopyAttachments(ctx, sentDBID, inboxDBID)
		}
	}

	// 5. Deliver to external recipients via SMTP
	if len(externalRecipients) > 0 {
		if p.smtpSender == nil {
			return fmt.Errorf("external delivery is disabled")
		}

		senderMailbox, err := p.mailboxRepo.GetByEmail(ctx, mailboxEmail)
		if err != nil || senderMailbox == nil {
			return fmt.Errorf("sender mailbox not found")
		}

		if err := p.smtpSender.Send(
			mailboxEmail,
			senderMailbox.DisplayName,
			externalRecipients,
			msg.Subject,
			msg.TextBody,
			msg.HTMLBody,
		); err != nil {
			return fmt.Errorf("failed to send via SMTP: %w", err)
		}
	}

	return nil
}

// MarkSeen updates the \Seen flag for a message.
func (p *DevDBMailProvider) MarkSeen(ctx context.Context, mailboxEmail string, folder string, messageID string, seen bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.msgRepo.UpdateSeen(ctx, mailboxEmail, messageID, seen)
}

// SetFlagged updates the flagged (starred) state of a message.
func (p *DevDBMailProvider) SetFlagged(ctx context.Context, mailboxEmail string, folder string, messageID string, flagged bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.msgRepo.UpdateFlagged(ctx, mailboxEmail, messageID, flagged)
}

// DeleteMessage moves a message to Trash.
func (p *DevDBMailProvider) DeleteMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.msgRepo.MoveToFolder(ctx, mailboxEmail, messageID, "Trash")
}

// SaveDraft stores an outgoing message as a draft.
func (p *DevDBMailProvider) SaveDraft(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	m := models.Message{
		ID:           generateUID(),
		MailboxEmail: mailboxEmail,
		Folder:       "Drafts",
		Subject:      msg.Subject,
		From:         mailboxEmail,
		To:           msg.To,
		Date:         time.Now(),
		TextBody:     msg.TextBody,
		HTMLBody:     msg.HTMLBody,
		Seen:         true,
		Draft:        true,
		InReplyTo:    msg.InReplyTo,
	}
	dbID, err := p.msgRepo.Create(ctx, &m)
	if err != nil {
		return err
	}
	for _, att := range msg.Attachments {
		_, _ = p.attRepo.Create(ctx, dbID, att.Filename, att.ContentType, att.Size, att.FilePath)
	}
	return nil
}

// EmptyTrash permanently deletes all messages in the Trash folder.
func (p *DevDBMailProvider) EmptyTrash(ctx context.Context, mailboxEmail string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.msgRepo.EmptyTrash(ctx, mailboxEmail)
}

// expireTrash removes messages older than trashTTLDays from Trash.
func (p *DevDBMailProvider) expireTrash(ctx context.Context, mailboxEmail string) error {
	cutoff := time.Now().AddDate(0, 0, -trashTTLDays)
	return p.msgRepo.ExpireTrash(ctx, mailboxEmail, cutoff)
}

func (p *DevDBMailProvider) loadAttachments(ctx context.Context, msg *models.Message) {
	if msg.DBID == 0 {
		return
	}
	atts, err := p.attRepo.ListByMessage(ctx, msg.DBID)
	if err == nil {
		msg.Attachments = atts
	}
}

func (p *DevDBMailProvider) resolveThreadID(ctx context.Context, inReplyTo string) string {
	if inReplyTo == "" {
		return generateUID()
	}
	threadID, _ := p.msgRepo.GetThreadIDByUID(ctx, inReplyTo)
	if threadID == "" {
		threadID = generateUID()
	}
	return threadID
}

func generateUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
