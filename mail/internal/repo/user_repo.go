// Package repo contains thin data-access layers that execute SQL and map
// results to domain models.  Each repo is responsible for exactly one
// aggregate root (user, mailbox, domain, etc.).
package repo

import (
	"context"
	"database/sql"

	"auto-hub/mail/internal/models"
)

// UserRepo performs CRUD operations on the users table.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo creates a new UserRepo backed by the given *sql.DB.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create inserts a new user and populates the ID, CreatedAt and UpdatedAt
// fields on the supplied model.
func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (email, password_hash, full_name, role, is_active, created_by_user_id)
			  VALUES ($1, $2, $3, $4, $5, $6)
			  RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		user.Email, user.PasswordHash, user.FullName, user.Role, btoi(user.IsActive), user.CreatedBy,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// GetByEmail looks up a user by their exact email address.  If no match is
// found nil is returned without an error.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, password_hash, full_name, role, is_active, created_by_user_id, created_at, updated_at
			  FROM users WHERE email = $1`
	row := r.db.QueryRowContext(ctx, query, email)
	u := &models.User{}
	var createdBy sql.NullInt64
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.IsActive, &createdBy, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if createdBy.Valid {
		v := int(createdBy.Int64)
		u.CreatedBy = &v
	}
	return u, nil
}

// GetByID fetches a single user by primary key.  Returns nil when the ID
// does not exist.
func (r *UserRepo) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT id, email, password_hash, full_name, role, is_active, created_by_user_id, created_at, updated_at
			  FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	u := &models.User{}
	var createdBy sql.NullInt64
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.IsActive, &createdBy, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if createdBy.Valid {
		v := int(createdBy.Int64)
		u.CreatedBy = &v
	}
	return u, nil
}

// List returns all users ordered by most recently created first.
func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	query := `SELECT id, email, password_hash, full_name, role, is_active, created_by_user_id, created_at, updated_at
			  FROM users ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var createdBy sql.NullInt64
		err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.IsActive, &createdBy, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if createdBy.Valid {
			v := int(createdBy.Int64)
			u.CreatedBy = &v
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Update persists changes to a user's profile fields (full_name, role,
// is_active).  It does not touch the password — use UpdatePassword for that.
func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET full_name = $1, role = $2, is_active = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4`,
		user.FullName, user.Role, btoi(user.IsActive), user.ID)
	return err
}

// UpdatePassword replaces the password_hash for a given user.
func (r *UserRepo) UpdatePassword(ctx context.Context, id int, hash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		hash, id)
	return err
}

// Delete permanently removes a user and, because of ON DELETE CASCADE,
// also cleans up their sessions and mailbox memberships.
func (r *UserRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
