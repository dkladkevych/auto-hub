package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"auto-hub/mail/internal/models"
)

// MessageRepo performs CRUD on the messages table.
type MessageRepo struct {
	db *sql.DB
}

// NewMessageRepo creates a new MessageRepo.
func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

// Create inserts a new message.
func (r *MessageRepo) Create(ctx context.Context, m *models.Message) (int64, error) {
	query := `INSERT INTO messages
				  (mailbox_email, folder, message_uid, in_reply_to, subject, sender_name, sender_email,
				   recipient, date, text_body, html_body, seen, flagged, thread_id, status)
				  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
				  RETURNING id`
	var id int64
	seen := 0
	if m.Seen {
		seen = 1
	}
	flagged := 0
	if m.Flagged {
		flagged = 1
	}
	err := r.db.QueryRowContext(ctx, query,
		m.MailboxEmail, m.Folder, m.ID, m.InReplyTo, m.Subject, "", m.From,
		m.To, m.Date.Format(time.RFC3339), m.TextBody, m.HTMLBody, seen, flagged,
		m.ThreadID, m.Status,
	).Scan(&id)
	return id, err
}

// ListByFolder returns paginated messages for a mailbox+folder.
func (r *MessageRepo) ListByFolder(ctx context.Context, mailboxEmail, folder string, limit, offset int) ([]models.Message, error) {
	query := `SELECT id, message_uid, folder, subject, sender_name, sender_email, recipient,
					 date, text_body, html_body, seen, flagged
				  FROM messages
				  WHERE mailbox_email = $1 AND folder = $2
				  ORDER BY date DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, mailboxEmail, folder, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Message
	for rows.Next() {
		var m models.Message
		var dateStr string
		var seen, flagged int
		err := rows.Scan(&m.DBID, &m.ID, &m.Folder, &m.Subject, &m.From, &m.From, &m.To,
			&dateStr, &m.TextBody, &m.HTMLBody, &seen, &flagged)
		if err != nil {
			return nil, err
		}
		m.Seen = seen == 1
		m.Flagged = flagged == 1
		m.Date, _ = time.Parse(time.RFC3339, dateStr)
		m.Snippet = firstLine(m.TextBody)
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetThreadIDByUID looks up the thread_id of any message with the given UID.
func (r *MessageRepo) GetThreadIDByUID(ctx context.Context, uid string) (string, error) {
	var threadID string
	err := r.db.QueryRowContext(ctx, `SELECT thread_id FROM messages WHERE message_uid = $1 LIMIT 1`, uid).Scan(&threadID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return threadID, nil
}

// CountByFolder returns the total message count for a folder.
func (r *MessageRepo) CountByFolder(ctx context.Context, mailboxEmail, folder string) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE mailbox_email = $1 AND folder = $2`
	var count int
	err := r.db.QueryRowContext(ctx, query, mailboxEmail, folder).Scan(&count)
	return count, err
}

// GetByUID returns a single message by its UID string.
func (r *MessageRepo) GetByUID(ctx context.Context, mailboxEmail, folder, uid string) (*models.Message, error) {
	query := `SELECT id, message_uid, folder, subject, sender_name, sender_email, recipient,
					 date, text_body, html_body, seen, flagged, thread_id, status
				  FROM messages
				  WHERE mailbox_email = $1 AND folder = $2 AND message_uid = $3`
	var m models.Message
	var dateStr string
	var seen, flagged int
	err := r.db.QueryRowContext(ctx, query, mailboxEmail, folder, uid).Scan(
		&m.DBID, &m.ID, &m.Folder, &m.Subject, &m.From, &m.From, &m.To,
		&dateStr, &m.TextBody, &m.HTMLBody, &seen, &flagged, &m.ThreadID, &m.Status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, err
	}
	m.Seen = seen == 1
	m.Flagged = flagged == 1
	m.Date, _ = time.Parse(time.RFC3339, dateStr)
	m.Snippet = firstLine(m.TextBody)
	return &m, nil
}

// UpdateSeen toggles the seen flag.
func (r *MessageRepo) UpdateSeen(ctx context.Context, mailboxEmail, uid string, seen bool) error {
	query := `UPDATE messages SET seen = $1 WHERE mailbox_email = $2 AND message_uid = $3`
	v := 0
	if seen {
		v = 1
	}
	_, err := r.db.ExecContext(ctx, query, v, mailboxEmail, uid)
	return err
}

// UpdateFlagged toggles the flagged (starred) flag.
func (r *MessageRepo) UpdateFlagged(ctx context.Context, mailboxEmail, uid string, flagged bool) error {
	query := `UPDATE messages SET flagged = $1 WHERE mailbox_email = $2 AND message_uid = $3`
	v := 0
	if flagged {
		v = 1
	}
	_, err := r.db.ExecContext(ctx, query, v, mailboxEmail, uid)
	return err
}

// MoveToFolder changes the folder of a message.
func (r *MessageRepo) MoveToFolder(ctx context.Context, mailboxEmail, uid, newFolder string) error {
	query := `UPDATE messages SET folder = $1 WHERE mailbox_email = $2 AND message_uid = $3`
	_, err := r.db.ExecContext(ctx, query, newFolder, mailboxEmail, uid)
	return err
}

// ListFlagged returns all flagged messages across folders.
func (r *MessageRepo) ListFlagged(ctx context.Context, mailboxEmail string, limit, offset int) ([]models.Message, error) {
	query := `SELECT id, message_uid, folder, subject, sender_name, sender_email, recipient,
					 date, text_body, html_body, seen, flagged
				  FROM messages
				  WHERE mailbox_email = $1 AND flagged = 1 AND folder != 'Trash'
				  ORDER BY date DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, mailboxEmail, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Message
	for rows.Next() {
		var m models.Message
		var dateStr string
		var seen, flagged int
		err := rows.Scan(&m.DBID, &m.ID, &m.Folder, &m.Subject, &m.From, &m.From, &m.To,
			&dateStr, &m.TextBody, &m.HTMLBody, &seen, &flagged)
		if err != nil {
			return nil, err
		}
		m.Seen = seen == 1
		m.Flagged = flagged == 1
		m.Date, _ = time.Parse(time.RFC3339, dateStr)
		m.Snippet = firstLine(m.TextBody)
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountUnseen returns the number of unseen messages in a folder.
func (r *MessageRepo) CountUnseen(ctx context.Context, mailboxEmail, folder string) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE mailbox_email = $1 AND folder = $2 AND seen = 0`
	var count int
	err := r.db.QueryRowContext(ctx, query, mailboxEmail, folder).Scan(&count)
	return count, err
}

// CountFlagged returns the number of flagged messages.
func (r *MessageRepo) CountFlagged(ctx context.Context, mailboxEmail string) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE mailbox_email = $1 AND flagged = 1 AND folder != 'Trash'`
	var count int
	err := r.db.QueryRowContext(ctx, query, mailboxEmail).Scan(&count)
	return count, err
}

// EmptyTrash permanently deletes Trash messages.
func (r *MessageRepo) EmptyTrash(ctx context.Context, mailboxEmail string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM messages WHERE mailbox_email = $1 AND folder = 'Trash'`,
		mailboxEmail,
	)
	return err
}

// ExpireTrash deletes messages in Trash older than cutoff.
func (r *MessageRepo) ExpireTrash(ctx context.Context, mailboxEmail string, cutoff time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM messages WHERE mailbox_email = $1 AND folder = 'Trash' AND created_at < $2`,
		mailboxEmail, cutoff.Format(time.RFC3339),
	)
	return err
}

// firstLine returns the first non-empty line of text for the snippet.
func firstLine(text string) string {
	lines := splitLines(text)
	for _, l := range lines {
		trimmed := trimSpace(l)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func trimSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	j := len(s)
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
