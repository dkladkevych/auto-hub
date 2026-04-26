package repo

import (
	"context"
	"database/sql"

	"auto-hub/mail/internal/models"
)

// SessionRepo performs CRUD operations on the sessions table.
type SessionRepo struct {
	db *sql.DB
}

// NewSessionRepo creates a new SessionRepo backed by the given *sql.DB.
func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create inserts a new session record and populates ID, CreatedAt and
// LastSeenAt on the supplied model.
func (r *SessionRepo) Create(ctx context.Context, s *models.Session) error {
	query := `INSERT INTO sessions (user_id, session_token_hash, user_agent, ip_address, expires_at)
			  VALUES ($1, $2, $3, $4, $5)
			  RETURNING id, created_at, last_seen_at`
	return r.db.QueryRowContext(ctx, query,
		s.UserID, s.SessionTokenHash, s.UserAgent, s.IPAddress, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt, &s.LastSeenAt)
}

// GetByTokenHash looks up a non-expired session by the SHA-256 hash of
// its raw token.  Returns nil when the session does not exist or has
// already expired.
func (r *SessionRepo) GetByTokenHash(ctx context.Context, hash string) (*models.Session, error) {
	query := `SELECT id, user_id, session_token_hash, user_agent, ip_address, expires_at, created_at, last_seen_at
			  FROM sessions WHERE session_token_hash = $1 AND expires_at > CURRENT_TIMESTAMP`
	row := r.db.QueryRowContext(ctx, query, hash)
	s := &models.Session{}
	err := row.Scan(&s.ID, &s.UserID, &s.SessionTokenHash, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt, &s.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Delete removes a session by its primary key.
func (r *SessionRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

// UpdateLastSeen bumps the last_seen_at timestamp to the current time.
func (r *SessionRepo) UpdateLastSeen(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	return err
}

// DeleteExpired removes all sessions whose expires_at is in the past.
// This should be run periodically (e.g. via a background goroutine or
// cron job) to prevent the table from growing indefinitely.
func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	return err
}
