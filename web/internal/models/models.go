package models

import "time"

// Listing represents a car listing in the inventory.
type Listing struct {
	ID           int
	AccountID    int
	Title        string
	Price        int
	Description  string
	SourceURL    *string
	Status       string
	Year         *int
	Make         *string
	Model        *string
	MileageKm    *int
	Location     *string
	Condition    *string
	Notes        *string
	Transmission *string
	Drivetrain   *string
	PublishedAt  *time.Time
}

// User represents a registered public user.
type User struct {
	ID                    int
	Email                 string
	PasswordHash          string
	FullName              *string
	IsVerified            bool
	VerificationCode      *string
	VerificationExpiresAt *time.Time
	CreatedAt             *time.Time
}

// Stats holds aggregated view counts.
type Stats struct {
	TargetType string
	TargetID   int
	ViewCount  int
}

// ViewLog holds individual view events for deduplication.
type ViewLog struct {
	ID           int
	TargetType   string
	TargetID     int
	Fingerprint  string
	ViewedDate   string
	ViewedAt     time.Time
}
