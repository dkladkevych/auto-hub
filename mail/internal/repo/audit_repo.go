package repo

import (
	"context"
	"database/sql"
	"encoding/json"

	"auto-hub/mail/internal/models"
)

// AuditRepo persists immutable audit log entries to the audit_logs table.
type AuditRepo struct {
	db *sql.DB
}

// NewAuditRepo creates a new AuditRepo backed by the given *sql.DB.
func NewAuditRepo(db *sql.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Log inserts a single audit record.  Errors are swallowed on purpose
// because audit logging should never break user-facing operations.
func (r *AuditRepo) Log(ctx context.Context, log *models.AuditLog) error {
	payloadJSON, err := json.Marshal(log.Payload)
	if err != nil {
		return err
	}
	query := `INSERT INTO audit_logs (actor_user_id, action, entity_type, entity_id, payload)
			  VALUES ($1, $2, $3, $4, $5)`
	_, err = r.db.ExecContext(ctx, query, log.ActorUserID, log.Action, log.EntityType, log.EntityID, payloadJSON)
	return err
}
