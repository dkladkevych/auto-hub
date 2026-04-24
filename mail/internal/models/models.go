// Package models contains the core data structures used across the mail
// control panel — users, domains, mailboxes, sessions and audit logs.
package models

import "time"

// User represents an account that can log in to the control panel or
// access mail via IMAP/Webmail.  A user with role "user" or "admin"
// always has an associated personal mailbox with the same email address.
type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedBy    *int      `json:"created_by_user_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Domain is a DNS domain managed by the control panel.  One domain can
// be marked as the default so that users do not have to select it
// manually when creating mailboxes.
type Domain struct {
	ID        int       `json:"id"`
	Domain    string    `json:"domain"`
	IsDefault bool      `json:"is_default"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Session stores a hashed authentication token tied to a browser cookie.
// The raw token is never persisted; only its SHA-256 digest is stored.
type Session struct {
	ID               int       `json:"id"`
	UserID           int       `json:"user_id"`
	SessionTokenHash string    `json:"-"`
	UserAgent        string    `json:"user_agent"`
	IPAddress        string    `json:"ip_address"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

// Mailbox is an email address that can receive and/or send messages.
// The MailboxType distinguishes personal (tied to a User), shared
// (team mailboxes with members), and system (programmatic) mailboxes.
type Mailbox struct {
	ID                  int       `json:"id"`
	Email               string    `json:"email"`
	LocalPart           string    `json:"local_part"`
	Domain              string    `json:"domain"`
	DisplayName         string    `json:"display_name"`
	MailboxType         string    `json:"mailbox_type"`
	IsActive            bool      `json:"is_active"`
	CanReceive          bool      `json:"can_receive"`
	CanSend             bool      `json:"can_send"`
	QuotaMb             int       `json:"quota_mb"`
	MaildirPath         string    `json:"maildir_path"`
	ImapLoginEnabled    bool      `json:"imap_login_enabled"`
	SmtpLoginEnabled    bool      `json:"smtp_login_enabled"`
	MailboxPasswordHash string    `json:"-"`
	CreatedBy           *int      `json:"created_by_user_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// MailboxMember links a User to a shared mailbox with a specific access
// role (read_only, user, or manager).
type MailboxMember struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	MailboxID  int       `json:"mailbox_id"`
	AccessRole string    `json:"access_role"`
	CreatedAt  time.Time `json:"created_at"`
	User       *User     `json:"user,omitempty"`
}

// AuditLog records mutating actions performed by users so that an
// administrator can review who changed what and when.
type AuditLog struct {
	ID          int                    `json:"id"`
	ActorUserID *int                   `json:"actor_user_id"`
	Action      string                 `json:"action"`
	EntityType  string                 `json:"entity_type"`
	EntityID    *int                   `json:"entity_id"`
	Payload     map[string]interface{} `json:"payload"`
	CreatedAt   time.Time              `json:"created_at"`
}
