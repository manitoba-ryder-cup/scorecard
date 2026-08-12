package golf

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SeedPlan is a tournament's advance setup with every name already resolved and every
// time already parsed — nothing in it can fail for a reason the setup file could have
// told us. Producing one is the caller's job; writing it is a single unit of work.
type SeedPlan struct {
	Tournament CreateTournamentInput
	CourseID   uuid.UUID
	TeeColorID uuid.UUID
	// Players is the entered roster, in file order. Each is matched to an existing player
	// by Email and created only if new, so someone who plays every year is one player.
	Players []SeedPlayer
	// Captains maps a team colour to a captain's lowercased email, which is one of Players.
	Captains map[string]string
	Matches  []PlannedMatch
}

// SeedPlayer is one entered player: their identity, and their attributes for this
// tournament. Tier is already defaulted.
type SeedPlayer struct {
	FirstName string
	LastName  string
	Email     string
	Tier      string
	Biography string
	Hdcp      float32
}

// PlannedMatch is one match with its format resolved and its tee time already an instant.
type PlannedMatch struct {
	Format   string
	FormatID uuid.UUID
	TeeTime  time.Time
}

// SeedSummary reports what a seed run created.
type SeedSummary struct {
	TournamentID   uuid.UUID
	PlayersEntered int
	Matches        int
}

// seedDB writes a whole tournament setup as one unit.
type seedDB interface {
	SeedTournament(ctx context.Context, plan SeedPlan) (*SeedSummary, error)
}

// SeedService creates a tournament's advance setup: the event, its two teams, the entered
// roster, each side's captain, and the match schedule.
//
// The draft and match participants are deliberately not here — those happen live at the
// event. Only the captains are drafted, because a side needs one before it can pick.
type SeedService struct {
	SeedDB seedDB
}

// Seed writes the planned setup. It either all lands or none of it does: a tournament
// half-created is worse than one not created, because a rerun then makes a second.
func (s *SeedService) Seed(ctx context.Context, plan SeedPlan) (*SeedSummary, error) {
	return s.SeedDB.SeedTournament(ctx, plan)
}
