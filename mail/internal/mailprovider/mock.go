// Package mailprovider defines the abstraction layer for email storage and transport.
package mailprovider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"auto-hub/mail/internal/models"
)

const trashTTLDays = 30

// MockMailProvider is an in-memory implementation of MailProvider for UI development.
// It is not a real mail engine — data lives only in memory and resets on restart.
type MockMailProvider struct {
	mu       sync.RWMutex
	folders  map[string][]models.Folder             // mailboxEmail -> folders
	messages map[string]map[string][]models.Message // mailboxEmail -> folder -> messages
	nextID   int
}

// NewMockMailProvider creates a mock provider pre-filled with fixture data
// for every mailbox found in the provided list.
func NewMockMailProvider(mailboxes []models.Mailbox) *MockMailProvider {
	m := &MockMailProvider{
		folders:  make(map[string][]models.Folder),
		messages: make(map[string]map[string][]models.Message),
		nextID:   1,
	}
	for _, mb := range mailboxes {
		m.seedMailbox(mb.Email)
	}
	return m
}

func (m *MockMailProvider) nextMessageID() string {
	m.nextID++
	return fmt.Sprintf("mock-%d-%d", time.Now().Unix(), m.nextID)
}

func (m *MockMailProvider) seedMailbox(email string) {
	folders := []models.Folder{
		{Name: "Inbox", Count: 4, Unseen: 2},
		{Name: "Sent", Count: 1, Unseen: 0},
		{Name: "Drafts", Count: 0, Unseen: 0},
		{Name: "Trash", Count: 0, Unseen: 0},
	}
	m.folders[email] = folders

	msgs := map[string][]models.Message{
		"Inbox": {
			{
				ID:       m.nextMessageID(),
				Folder:   "Inbox",
				Subject:  "Welcome to Auto-Hub Mail",
				From:     "support@auto-hub.ca",
				To:       email,
				Date:     time.Now().Add(-48 * time.Hour),
				Snippet:  "Your mailbox has been configured successfully...",
				TextBody: "Your mailbox has been configured successfully. You can start sending and receiving emails right away.\n\nBest regards,\nAuto-Hub Team",
				HTMLBody: "<p>Your mailbox has been configured successfully. You can start sending and receiving emails right away.</p><p>Best regards,<br>Auto-Hub Team</p>",
				Seen:     true,
			},
			{
				ID:       m.nextMessageID(),
				Folder:   "Inbox",
				Subject:  "Meeting reminder: Project Sync",
				From:     "alice@auto-hub.ca",
				To:       email,
				Date:     time.Now().Add(-24 * time.Hour),
				Snippet:  "Don't forget about the project sync at 2 PM today...",
				TextBody: "Hi,\n\nDon't forget about the project sync at 2 PM today. We'll discuss the roadmap for Q3.\n\nCheers,\nAlice",
				HTMLBody: "<p>Hi,</p><p>Don't forget about the project sync at <strong>2 PM today</strong>. We'll discuss the roadmap for Q3.</p><p>Cheers,<br>Alice</p>",
				Seen:     false,
			},
			{
				ID:       m.nextMessageID(),
				Folder:   "Inbox",
				Subject:  "Invoice #1024",
				From:     "billing@example.com",
				To:       email,
				Cc:       "accounting@auto-hub.ca",
				Date:     time.Now().Add(-12 * time.Hour),
				Snippet:  "Please find attached the invoice for services rendered...",
				TextBody: "Please find attached the invoice for services rendered in March.\n\nTotal: $499.00\n\nLet us know if you have any questions.",
				HTMLBody: "<p>Please find attached the invoice for services rendered in March.</p><p>Total: <strong>$499.00</strong></p><p>Let us know if you have any questions.</p>",
				Seen:     false,
				Flagged:  true,
				Attachments: []models.Attachment{
					{Filename: "invoice-1024.pdf", ContentType: "application/pdf", Size: 42000},
				},
			},
			{
				ID:       m.nextMessageID(),
				Folder:   "Inbox",
				Subject:  "Re: Support ticket #42",
				From:     "bob@auto-hub.ca",
				To:       email,
				Date:     time.Now().Add(-2 * time.Hour),
				Snippet:  "The issue has been resolved on our end...",
				TextBody: "The issue has been resolved on our end. Please verify and let us know if you need anything else.",
				HTMLBody: "<p>The issue has been resolved on our end. Please verify and let us know if you need anything else.</p>",
				Seen:     true,
				Answered: true,
			},
		},
		"Sent": {
			{
				ID:       m.nextMessageID(),
				Folder:   "Sent",
				Subject:  "Re: Meeting reminder: Project Sync",
				From:     email,
				To:       "alice@auto-hub.ca",
				Date:     time.Now().Add(-23 * time.Hour),
				Snippet:  "Thanks for the reminder, I'll be there...",
				TextBody: "Thanks for the reminder, I'll be there.\n\nSee you soon.",
				HTMLBody: "<p>Thanks for the reminder, I'll be there.</p><p>See you soon.</p>",
				Seen:     true,
			},
		},
		"Drafts": {},
		"Trash":  {},
	}
	m.messages[email] = msgs
}

func (m *MockMailProvider) ensureMailbox(email string) {
	if _, ok := m.messages[email]; !ok {
		m.seedMailbox(email)
	}
}

// ListFolders returns all folders for the given mailbox email address.
func (m *MockMailProvider) ListFolders(ctx context.Context, mailboxEmail string) ([]models.Folder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	m.expireTrash(mailboxEmail)
	folders := m.folders[mailboxEmail]
	out := make([]models.Folder, len(folders))
	copy(out, folders)
	return out, nil
}

// ListMessages returns messages in a folder with simple pagination.
func (m *MockMailProvider) ListMessages(ctx context.Context, mailboxEmail string, folder string, limit, offset int) ([]models.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	if folder == "Trash" {
		m.expireTrash(mailboxEmail)
	}
	mailboxMsgs := m.messages[mailboxEmail]
	msgs, ok := mailboxMsgs[folder]
	if !ok {
		return nil, nil
	}
	if offset >= len(msgs) {
		return nil, nil
	}
	end := offset + limit
	if end > len(msgs) || limit <= 0 {
		end = len(msgs)
	}
	out := make([]models.Message, end-offset)
	copy(out, msgs[offset:end])
	if folder == "Trash" {
		for i := range out {
			if out[i].DeletedAt != nil {
				elapsed := time.Since(*out[i].DeletedAt).Hours() / 24
				left := trashTTLDays - int(elapsed)
				if left < 0 {
					left = 0
				}
				out[i].TrashDaysLeft = left
			} else {
				out[i].TrashDaysLeft = trashTTLDays
			}
		}
	}
	return out, nil
}

// GetMessage returns a single message by its ID within a folder.
func (m *MockMailProvider) GetMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) (*models.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	mailboxMsgs := m.messages[mailboxEmail]
	msgs, ok := mailboxMsgs[folder]
	if !ok {
		return nil, fmt.Errorf("folder not found")
	}
	for i := range msgs {
		if msgs[i].ID == messageID {
			msg := msgs[i]
			return &msg, nil
		}
	}
	return nil, fmt.Errorf("message not found")
}

// SendMessage sends an outgoing message.
func (m *MockMailProvider) SendMessage(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	mailboxMsgs := m.messages[mailboxEmail]
	sent := models.Message{
		ID:       m.nextMessageID(),
		Folder:   "Sent",
		Subject:  msg.Subject,
		From:     mailboxEmail,
		To:       msg.To,
		Cc:       msg.Cc,
		Date:     time.Now(),
		Snippet:  firstLine(msg.TextBody),
		TextBody: msg.TextBody,
		HTMLBody: msg.HTMLBody,
		Seen:     true,
	}
	mailboxMsgs["Sent"] = append(mailboxMsgs["Sent"], sent)
	m.updateFolderCounts(mailboxEmail)
	return nil
}

// MarkSeen updates the \\Seen flag for a message.
func (m *MockMailProvider) MarkSeen(ctx context.Context, mailboxEmail string, folder string, messageID string, seen bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	mailboxMsgs := m.messages[mailboxEmail]
	msgs, ok := mailboxMsgs[folder]
	if !ok {
		return fmt.Errorf("folder not found")
	}
	for i := range msgs {
		if msgs[i].ID == messageID {
			msgs[i].Seen = seen
			m.updateFolderCounts(mailboxEmail)
			return nil
		}
	}
	return fmt.Errorf("message not found")
}

// DeleteMessage removes a message from a folder.
func (m *MockMailProvider) DeleteMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	mailboxMsgs := m.messages[mailboxEmail]
	msgs, ok := mailboxMsgs[folder]
	if !ok {
		return fmt.Errorf("folder not found")
	}
	for i, msg := range msgs {
		if msg.ID == messageID {
			now := time.Now()
			msg.Folder = "Trash"
			msg.DeletedAt = &now
			mailboxMsgs[folder] = append(msgs[:i], msgs[i+1:]...)
			mailboxMsgs["Trash"] = append(mailboxMsgs["Trash"], msg)
			m.updateFolderCounts(mailboxEmail)
			return nil
		}
	}
	return fmt.Errorf("message not found")
}

// EmptyTrash permanently deletes all messages in the Trash folder.
func (m *MockMailProvider) EmptyTrash(ctx context.Context, mailboxEmail string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	mailboxMsgs := m.messages[mailboxEmail]
	mailboxMsgs["Trash"] = nil
	m.updateFolderCounts(mailboxEmail)
	return nil
}

// SaveDraft stores an outgoing message as a draft.
func (m *MockMailProvider) SaveDraft(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureMailbox(mailboxEmail)
	mailboxMsgs := m.messages[mailboxEmail]
	draft := models.Message{
		ID:       m.nextMessageID(),
		Folder:   "Drafts",
		Subject:  msg.Subject,
		From:     mailboxEmail,
		To:       msg.To,
		Cc:       msg.Cc,
		Date:     time.Now(),
		Snippet:  firstLine(msg.TextBody),
		TextBody: msg.TextBody,
		HTMLBody: msg.HTMLBody,
		Seen:     true,
		Draft:    true,
	}
	mailboxMsgs["Drafts"] = append(mailboxMsgs["Drafts"], draft)
	m.updateFolderCounts(mailboxEmail)
	return nil
}

// expireTrash removes messages older than trashTTLDays from Trash.
func (m *MockMailProvider) expireTrash(mailboxEmail string) {
	mailboxMsgs := m.messages[mailboxEmail]
	trash := mailboxMsgs["Trash"]
	cutoff := time.Now().AddDate(0, 0, -trashTTLDays)
	var kept []models.Message
	for _, msg := range trash {
		if msg.DeletedAt != nil && msg.DeletedAt.Before(cutoff) {
			continue
		}
		kept = append(kept, msg)
	}
	mailboxMsgs["Trash"] = kept
	m.updateFolderCounts(mailboxEmail)
}

func (m *MockMailProvider) updateFolderCounts(mailboxEmail string) {
	mailboxMsgs := m.messages[mailboxEmail]
	folders := m.folders[mailboxEmail]
	for i := range folders {
		msgs := mailboxMsgs[folders[i].Name]
		folders[i].Count = len(msgs)
		unseen := 0
		for _, msg := range msgs {
			if !msg.Seen {
				unseen++
			}
		}
		folders[i].Unseen = unseen
	}
}

func firstLine(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}
