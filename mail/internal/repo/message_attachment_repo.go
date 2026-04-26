package repo

import (
	"context"
	"database/sql"

	"auto-hub/mail/internal/models"
)

// MessageAttachmentRepo manages file attachments linked to messages.
type MessageAttachmentRepo struct {
	db *sql.DB
}

// NewMessageAttachmentRepo creates a new MessageAttachmentRepo.
func NewMessageAttachmentRepo(db *sql.DB) *MessageAttachmentRepo {
	return &MessageAttachmentRepo{db: db}
}

// Create inserts an attachment record.
func (r *MessageAttachmentRepo) Create(ctx context.Context, messageID int64, filename, contentType string, fileSize int64, filePath string) (int64, error) {
	query := `INSERT INTO message_attachments
			  (message_id, filename, content_type, file_size, file_path)
			  VALUES ($1,$2,$3,$4,$5)
			  RETURNING id`
	var id int64
	err := r.db.QueryRowContext(ctx, query, messageID, filename, contentType, fileSize, filePath).Scan(&id)
	return id, err
}

// ListByMessage returns all attachments for a given message database ID.
func (r *MessageAttachmentRepo) ListByMessage(ctx context.Context, messageID int64) ([]models.Attachment, error) {
	query := `SELECT filename, content_type, file_size, file_path FROM message_attachments WHERE message_id = $1`
	rows, err := r.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Attachment
	for rows.Next() {
		var a models.Attachment
		err := rows.Scan(&a.Filename, &a.ContentType, &a.Size, &a.FilePath)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CopyAttachments duplicates attachment records from one message to another.
func (r *MessageAttachmentRepo) CopyAttachments(ctx context.Context, fromMessageID, toMessageID int64) error {
	atts, err := r.ListByMessage(ctx, fromMessageID)
	if err != nil {
		return err
	}
	for _, a := range atts {
		_, _ = r.Create(ctx, toMessageID, a.Filename, a.ContentType, a.Size, a.FilePath)
	}
	return nil
}
