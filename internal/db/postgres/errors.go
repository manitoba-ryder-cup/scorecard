package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// PostgreSQL SQLSTATE codes we translate to domain sentinels.
const (
	pgUniqueViolation     = "23505" // duplicate key
	pgForeignKeyViolation = "23503" // referenced row does not exist
)

// mapReadErr translates a single-row read error into a domain sentinel: the driver's
// no-rows error becomes golf.ErrNotFound so the API returns 404 instead of 500. Only
// for queries where a missing row means "not found" — not optional-row lookups, which
// handle pgx.ErrNoRows themselves by returning nil.
func mapReadErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return golf.ErrNotFound
	}
	return err
}

// mapWriteErr translates a database write error into a domain sentinel where one
// applies, keeping driver types out of the domain and API. A unique violation (e.g.
// a second Red team, or a duplicate tournament) becomes golf.ErrConflict; everything
// else passes through unchanged.
func mapWriteErr(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("%w: %s", golf.ErrConflict, pgErr.ConstraintName)
		case pgForeignKeyViolation:
			// A body field references a row that doesn't exist (e.g. an unknown
			// tee_color_id or player_id) — a client error, not a server fault.
			return fmt.Errorf("%w: %s", golf.ErrInvalidInput, pgErr.ConstraintName)
		}
	}
	return err
}
