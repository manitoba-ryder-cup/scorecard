package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// This file maps between the database driver's nullable types and the plain
// stdlib types the domain uses (the domain never sees pgtype).

// pgDateToTime converts a pgtype.Date to time.Time (zero time when not set).
func pgDateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

// timeToPgDate converts a time.Time to a pgtype.Date (invalid/NULL for the zero time).
func timeToPgDate(t time.Time) pgtype.Date {
	if t.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: t, Valid: true}
}

// pgTimestampToPtr converts a nullable pgtype.Timestamp to *time.Time.
func pgTimestampToPtr(ts pgtype.Timestamp) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
