package repo

import (
	"context"
	"database/sql"

	"auto-hub/mail/internal/models"
)

// MailboxRepo performs CRUD operations on the mailboxes table.
type MailboxRepo struct {
	db *sql.DB
}

// NewMailboxRepo creates a new MailboxRepo backed by the given *sql.DB.
func NewMailboxRepo(db *sql.DB) *MailboxRepo {
	return &MailboxRepo{db: db}
}

// Create inserts a new mailbox record.  The ID, CreatedAt and UpdatedAt
// fields are populated on the supplied model.
func (r *MailboxRepo) Create(ctx context.Context, m *models.Mailbox) error {
	query := `INSERT INTO mailboxes
			  (email, local_part, domain, display_name, mailbox_type, is_active,
			   can_receive, can_send, quota_mb, maildir_path,
			   imap_login_enabled, smtp_login_enabled, mailbox_password_hash, created_by_user_id)
			  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			  RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		m.Email, m.LocalPart, m.Domain, m.DisplayName, m.MailboxType, m.IsActive,
		m.CanReceive, m.CanSend, m.QuotaMb, m.MaildirPath,
		m.ImapLoginEnabled, m.SmtpLoginEnabled, m.MailboxPasswordHash, m.CreatedBy,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

// GetByID fetches a single mailbox by primary key.
func (r *MailboxRepo) GetByID(ctx context.Context, id int) (*models.Mailbox, error) {
	query := `SELECT id, email, local_part, domain, display_name, mailbox_type, is_active,
			     can_receive, can_send, quota_mb, maildir_path,
			     imap_login_enabled, smtp_login_enabled, mailbox_password_hash,
			     created_by_user_id, created_at, updated_at
			  FROM mailboxes WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	m := &models.Mailbox{}
	var createdBy sql.NullInt64
	err := row.Scan(&m.ID, &m.Email, &m.LocalPart, &m.Domain, &m.DisplayName, &m.MailboxType,
		&m.IsActive, &m.CanReceive, &m.CanSend, &m.QuotaMb, &m.MaildirPath,
		&m.ImapLoginEnabled, &m.SmtpLoginEnabled, &m.MailboxPasswordHash,
		&createdBy, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if createdBy.Valid {
		v := int(createdBy.Int64)
		m.CreatedBy = &v
	}
	return m, nil
}

// List returns every mailbox in the database, ordered by creation time.
func (r *MailboxRepo) List(ctx context.Context) ([]models.Mailbox, error) {
	query := `SELECT id, email, local_part, domain, display_name, mailbox_type, is_active,
			     can_receive, can_send, quota_mb, maildir_path,
			     imap_login_enabled, smtp_login_enabled, mailbox_password_hash,
			     created_by_user_id, created_at, updated_at
			  FROM mailboxes ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mailboxes []models.Mailbox
	for rows.Next() {
		var m models.Mailbox
		var createdBy sql.NullInt64
		err := rows.Scan(&m.ID, &m.Email, &m.LocalPart, &m.Domain, &m.DisplayName, &m.MailboxType,
			&m.IsActive, &m.CanReceive, &m.CanSend, &m.QuotaMb, &m.MaildirPath,
			&m.ImapLoginEnabled, &m.SmtpLoginEnabled, &m.MailboxPasswordHash,
			&createdBy, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if createdBy.Valid {
			v := int(createdBy.Int64)
			m.CreatedBy = &v
		}
		mailboxes = append(mailboxes, m)
	}
	return mailboxes, rows.Err()
}

// Update persists changes to a mailbox's display name, type and quota.
func (r *MailboxRepo) Update(ctx context.Context, m *models.Mailbox) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mailboxes SET display_name = $1, mailbox_type = $2, quota_mb = $3, updated_at = datetime('now') WHERE id = $4`,
		m.DisplayName, m.MailboxType, m.QuotaMb, m.ID)
	return err
}

// UpdateSettings persists the can_receive and can_send flags.
func (r *MailboxRepo) UpdateSettings(ctx context.Context, mailboxID int, canReceive, canSend bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mailboxes SET can_receive = $1, can_send = $2, updated_at = datetime('now') WHERE id = $3`,
		canReceive, canSend, mailboxID)
	return err
}

// UpdatePassword replaces the mailbox_password_hash for a given mailbox.
func (r *MailboxRepo) UpdatePassword(ctx context.Context, mailboxID int, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mailboxes SET mailbox_password_hash = $1, updated_at = datetime('now') WHERE id = $2`,
		hash, mailboxID)
	return err
}

// Delete permanently removes a mailbox by ID.
func (r *MailboxRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mailboxes WHERE id = $1`, id)
	return err
}

// CountByDomain returns how many mailboxes belong to a specific domain.
// This is used to block domain deletion when mailboxes still exist.
func (r *MailboxRepo) CountByDomain(ctx context.Context, domain string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mailboxes WHERE domain = $1`, domain).Scan(&count)
	return count, err
}
