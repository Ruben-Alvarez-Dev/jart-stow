package sqlite

import (
	"database/sql"
	"time"
)

// nullableTime converts a *time.Time to sql.NullString (RFC3339 format) for SQLite storage.
func nullableTime(t *time.Time) sql.NullString {
	if t == nil || t.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339), Valid: true}
}

// nullableString converts a string to sql.NullString. Empty strings become NULL.
func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
