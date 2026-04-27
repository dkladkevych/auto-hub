package repo

import (
	"database/sql"
	"time"

	"auto-hub/web/internal/models"
)

// UserRepo provides data access for users.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func scanUser(row *sql.Row) (*models.User, error) {
	var u models.User
	var fullName, code sql.NullString
	var expiresAt, createdAt sql.NullTime

	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &fullName,
		&u.IsVerified, &code, &expiresAt, &createdAt)
	if err != nil {
		return nil, err
	}
	u.FullName = nullStringPtr(fullName)
	u.VerificationCode = nullStringPtr(code)
	u.VerificationExpiresAt = nullTimePtr(expiresAt)
	u.CreatedAt = nullTimePtr(createdAt)
	return &u, nil
}

// Create inserts a new user and returns the ID.
func (r *UserRepo) Create(email, passwordHash, fullName string, code string, expiresAt time.Time) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO users (email, password_hash, full_name, verification_code, verification_expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		email, passwordHash, fullName, code, expiresAt,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetByEmail fetches a user by email.
func (r *UserRepo) GetByEmail(email string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, password_hash, full_name, is_verified,
			verification_code, verification_expires_at, created_at
		FROM users WHERE email = ?`, email)
	return scanUser(row)
}

// GetByID fetches a user by ID.
func (r *UserRepo) GetByID(id int) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, password_hash, full_name, is_verified,
			verification_code, verification_expires_at, created_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// Verify confirms a user's email by code.
func (r *UserRepo) Verify(email, code string) (bool, error) {
	res, err := r.db.Exec(`
		UPDATE users
		SET is_verified = 1, verification_code = NULL, verification_expires_at = NULL
		WHERE email = ? AND verification_code = ?
		  AND verification_expires_at > datetime('now')`,
		email, code,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// GetVerifiedByEmail fetches only verified users.
func (r *UserRepo) GetVerifiedByEmail(email string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, password_hash, full_name, is_verified,
			verification_code, verification_expires_at, created_at
		FROM users WHERE email = ? AND is_verified = 1`, email)
	return scanUser(row)
}
