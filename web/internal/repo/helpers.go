package repo

import (
	"database/sql"
	"time"
)

func nullIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func nullStringPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	return &n.String
}

func nullTimePtr(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}

func ptrInt(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}

func ptrString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *v, Valid: true}
}

func ptrTime(v *time.Time) sql.NullTime {
	if v == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *v, Valid: true}
}
