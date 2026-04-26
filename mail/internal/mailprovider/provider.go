// Package mailprovider defines the abstraction layer for email storage and transport.
package mailprovider

import (
	"context"

	"auto-hub/mail/internal/models"
)

// MailProvider abstracts email storage and transport operations.
// UI code depends only on this interface, allowing future IMAP/SMTP
// implementations without changes to handlers or templates.
type MailProvider interface {
	// ListFolders returns all folders for the given mailbox email address.
	ListFolders(ctx context.Context, mailboxEmail string) ([]models.Folder, error)

	// ListMessages returns messages in a folder with simple pagination.
	ListMessages(ctx context.Context, mailboxEmail string, folder string, limit, offset int) ([]models.Message, error)

	// CountMessages returns the total number of messages in a folder.
	CountMessages(ctx context.Context, mailboxEmail string, folder string) (int, error)

	// GetMessage returns a single message by its ID within a folder.
	GetMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) (*models.Message, error)

	// SendMessage sends an outgoing message.
	SendMessage(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error

	// MarkSeen updates the \\Seen flag for a message.
	MarkSeen(ctx context.Context, mailboxEmail string, folder string, messageID string, seen bool) error
	// SetFlagged updates the flagged (starred) state of a message.
	SetFlagged(ctx context.Context, mailboxEmail string, folder string, messageID string, flagged bool) error

	// DeleteMessage removes a message from a folder.
	DeleteMessage(ctx context.Context, mailboxEmail string, folder string, messageID string) error

	// SaveDraft stores an outgoing message as a draft.
	SaveDraft(ctx context.Context, mailboxEmail string, msg *models.OutgoingMessage) error

	// EmptyTrash permanently deletes all messages in the Trash folder.
	EmptyTrash(ctx context.Context, mailboxEmail string) error
}
