package golf

import (
	"errors"
	"fmt"
)

// Sentinel errors let the API layer translate domain failures to HTTP status codes without
// knowing the domain's internals. Their strings are developer-facing: they reach the log
// through the wrapped chain, and the sentence a caller reads is chosen at the API boundary.
var (
	// ErrInvalidInput marks a caller-supplied value that failed domain validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrConflict marks a write that collides with an existing row (e.g. a duplicate
	// team color or tournament). Repositories translate the database's unique-violation
	// into this so the domain and API stay driver-agnostic.
	ErrConflict = errors.New("resource conflict")
	// ErrNotFound marks a requested resource that does not exist (or is invisible to
	// the tenant). Repositories translate the driver's no-rows error into this so the
	// API can return 404 rather than mistaking a missing row for an internal failure.
	ErrNotFound = errors.New("resource not found")
)

// Each names the row a read did not find, so the API can say which one. They wrap
// ErrNotFound, so a caller that only needs "this is a 404" still matches on that.
var (
	ErrCourseNotFound           = fmt.Errorf("%w: course", ErrNotFound)
	ErrMatchNotFound            = fmt.Errorf("%w: match", ErrNotFound)
	ErrParticipantNotFound      = fmt.Errorf("%w: match participant", ErrNotFound)
	ErrPlayerNotFound           = fmt.Errorf("%w: player", ErrNotFound)
	ErrTeamMemberNotFound       = fmt.Errorf("%w: team member", ErrNotFound)
	ErrTeamNotFound             = fmt.Errorf("%w: team", ErrNotFound)
	ErrTournamentNotFound       = fmt.Errorf("%w: tournament", ErrNotFound)
	ErrTournamentPlayerNotFound = fmt.Errorf("%w: tournament player", ErrNotFound)
)

// Each names a write the scoring rule refused. Separate sentinels because the way through
// differs: what a caller has to reset, and what they can do afterwards, is not the same.
var (
	ErrScoredMatchDelete   = fmt.Errorf("%w: match is scored, cannot delete", ErrConflict)
	ErrScoredMatchLineup   = fmt.Errorf("%w: match is scored, cannot change lineup", ErrConflict)
	ErrScoredMatchTeeSet   = fmt.Errorf("%w: match is scored, cannot move tee set", ErrConflict)
	ErrScoredPlayerUndraft = fmt.Errorf("%w: player has scored matches, cannot undraft", ErrConflict)
)
