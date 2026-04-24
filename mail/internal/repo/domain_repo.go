package repo

import (
	"context"
	"database/sql"

	"auto-hub/mail/internal/models"
)

// DomainRepo performs CRUD operations on the domains table.
type DomainRepo struct {
	db *sql.DB
}

// NewDomainRepo creates a new DomainRepo backed by the given *sql.DB.
func NewDomainRepo(db *sql.DB) *DomainRepo {
	return &DomainRepo{db: db}
}

// Create inserts a new domain and populates ID and CreatedAt on the model.
func (r *DomainRepo) Create(ctx context.Context, d *models.Domain) error {
	query := `INSERT INTO domains (domain, is_default, is_active) VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, d.Domain, d.IsDefault, d.IsActive).Scan(&d.ID, &d.CreatedAt)
}

// GetByID looks up a domain by its primary key.
func (r *DomainRepo) GetByID(ctx context.Context, id int) (*models.Domain, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, domain, is_default, is_active, created_at FROM domains WHERE id = $1`, id)
	d := &models.Domain{}
	err := row.Scan(&d.ID, &d.Domain, &d.IsDefault, &d.IsActive, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// GetDefault returns the single domain marked as default and active.
// If none is configured, nil is returned.
func (r *DomainRepo) GetDefault(ctx context.Context) (*models.Domain, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, domain, is_default, is_active, created_at FROM domains WHERE is_default = 1 AND is_active = 1 LIMIT 1`)
	d := &models.Domain{}
	err := row.Scan(&d.ID, &d.Domain, &d.IsDefault, &d.IsActive, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// ListActive returns only the domains that are currently active, ordered
// with the default domain first.
func (r *DomainRepo) ListActive(ctx context.Context) ([]models.Domain, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, domain, is_default, is_active, created_at FROM domains WHERE is_active = 1 ORDER BY is_default DESC, domain`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []models.Domain
	for rows.Next() {
		var d models.Domain
		if err := rows.Scan(&d.ID, &d.Domain, &d.IsDefault, &d.IsActive, &d.CreatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

// ListAll returns every domain including inactive ones.
func (r *DomainRepo) ListAll(ctx context.Context) ([]models.Domain, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, domain, is_default, is_active, created_at FROM domains ORDER BY is_default DESC, domain`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []models.Domain
	for rows.Next() {
		var d models.Domain
		if err := rows.Scan(&d.ID, &d.Domain, &d.IsDefault, &d.IsActive, &d.CreatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

// ClearDefault unsets the default flag on ALL domains.  This is called
// before setting a new default so that exactly one domain is default at
// any time.
func (r *DomainRepo) ClearDefault(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `UPDATE domains SET is_default = 0`)
	return err
}

// SetDefault marks a specific domain as the default.
func (r *DomainRepo) SetDefault(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE domains SET is_default = 1 WHERE id = $1`, id)
	return err
}

// Delete permanently removes a domain by ID.
func (r *DomainRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM domains WHERE id = $1`, id)
	return err
}
