package repo

import (
	"context"
	"database/sql"

	"auto-hub/mail/internal/models"
)

// MailboxMemberRepo performs CRUD operations on the mailbox_members
// join table that links users to shared mailboxes.
type MailboxMemberRepo struct {
	db *sql.DB
}

// NewMailboxMemberRepo creates a new MailboxMemberRepo backed by the
// given *sql.DB.
func NewMailboxMemberRepo(db *sql.DB) *MailboxMemberRepo {
	return &MailboxMemberRepo{db: db}
}

// Add inserts a new membership record.
func (r *MailboxMemberRepo) Add(ctx context.Context, m *models.MailboxMember) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO mailbox_members (user_id, mailbox_id, access_role) VALUES ($1, $2, $3) RETURNING id, created_at`,
		m.UserID, m.MailboxID, m.AccessRole,
	).Scan(&m.ID, &m.CreatedAt)
}

// GetByID fetches a single membership by its primary key.
func (r *MailboxMemberRepo) GetByID(ctx context.Context, id int) (*models.MailboxMember, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, mailbox_id, access_role, created_at FROM mailbox_members WHERE id = $1`, id)
	m := &models.MailboxMember{}
	err := row.Scan(&m.ID, &m.UserID, &m.MailboxID, &m.AccessRole, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// Exists returns true when the given user is already a member of the
// specified mailbox.
func (r *MailboxMemberRepo) Exists(ctx context.Context, userID, mailboxID int) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM mailbox_members WHERE user_id = $1 AND mailbox_id = $2`,
		userID, mailboxID).Scan(&count)
	return count > 0, err
}

// ListByMailbox returns every member of a mailbox together with the
// associated user data (JOIN with the users table).
func (r *MailboxMemberRepo) ListByMailbox(ctx context.Context, mailboxID int) ([]models.MailboxMember, error) {
	query := `SELECT m.id, m.user_id, m.mailbox_id, m.access_role, m.created_at,
			     u.id, u.email, u.password_hash, u.full_name, u.role, u.is_active, u.created_by_user_id, u.created_at, u.updated_at
			  FROM mailbox_members m
			  JOIN users u ON m.user_id = u.id
			  WHERE m.mailbox_id = $1`
	rows, err := r.db.QueryContext(ctx, query, mailboxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.MailboxMember
	for rows.Next() {
		var m models.MailboxMember
		var u models.User
		var createdBy sql.NullInt64
		err := rows.Scan(&m.ID, &m.UserID, &m.MailboxID, &m.AccessRole, &m.CreatedAt,
			&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.IsActive, &createdBy, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if createdBy.Valid {
			v := int(createdBy.Int64)
			u.CreatedBy = &v
		}
		m.User = &u
		members = append(members, m)
	}
	return members, rows.Err()
}

// Remove deletes a membership by its primary key.
func (r *MailboxMemberRepo) Remove(ctx context.Context, memberID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mailbox_members WHERE id = $1`, memberID)
	return err
}
