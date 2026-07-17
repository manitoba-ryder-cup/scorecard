package golf

import (
	"time"

	"github.com/google/uuid"
)

// The domain layer holds no tenant_id — tenancy is a persistence/RLS concern
// carried in context by the repositories, never a domain field. Entity IDs are all
// uuid.UUID (matching the schema); non-ID integers (hole number, par, strokes) stay
// int. Dates are plain time.Time; the repos map to/from the database driver's types.

// Player represents a golfer's stable identity. Per-tournament attributes (tier,
// biography, handicap) live on TournamentPlayer; win/loss records are derived from
// match_results.
type Player struct {
	ID        uuid.UUID
	UserID    *uuid.UUID // heimdall account link; nil for roster-only players
	Email     *string
	FirstName string
	LastName  string
	PhotoPath string
}

// Team represents one of a tournament's two sides.
type Team struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	Color        string
	CaptainID    *uuid.UUID
}

// Tournament represents a golf tournament event
type Tournament struct {
	ID        uuid.UUID
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Location  string
}

// Match represents an individual golf match
type Match struct {
	ID            uuid.UUID
	TournamentID  uuid.UUID
	CourseID      uuid.UUID
	TeeColorID    uuid.UUID
	MatchFormatID uuid.UUID
	TeeTime       *time.Time
	Handicapped   bool
}

// MatchParticipant links a player (on a team) to a match.
type MatchParticipant struct {
	TournamentID uuid.UUID
	MatchID      uuid.UUID
	PlayerID     uuid.UUID
	TeamID       uuid.UUID
}

// Score is a hole score attributed to a side (TeamID) and, for per-player formats,
// to a player. PlayerID is nil for one-ball team scores (alt shot, scramble).
type Score struct {
	ID         uuid.UUID
	MatchID    uuid.UUID
	TeamID     uuid.UUID
	PlayerID   *uuid.UUID
	CourseID   uuid.UUID
	TeeColorID uuid.UUID
	HoleNumber int32
	Strokes    int32
}

// Course represents a golf course
type Course struct {
	ID   uuid.UUID
	Name string
}

// Hole represents a hole on a golf course with specific tee
type Hole struct {
	CourseID   uuid.UUID
	TeeColorID uuid.UUID
	Number     int32
	Par        int32
	Hdcp       int32
	Yards      int32
}

// TeeColor represents tee marker colors
type TeeColor struct {
	ID    uuid.UUID
	Color string
}

// TeeSet represents course rating and slope for a specific tee
type TeeSet struct {
	CourseID   uuid.UUID
	TeeColorID uuid.UUID
	Slope      int32
	Rating     float64
}

// MatchFormat represents the type of match
type MatchFormat struct {
	ID   uuid.UUID
	Name string
}

// TeamMember is the draft outcome: a player assigned to a team for a tournament.
// Per-tournament attributes live on TournamentPlayer, not here.
type TeamMember struct {
	TeamID       uuid.UUID
	PlayerID     uuid.UUID
	TournamentID uuid.UUID
}

// TournamentPlayer is a player entered in a tournament: the per-tournament attributes
// (tier, biography, handicap) set independently of the team draft, plus the player's
// identity and their team assignment. TeamID is nil when entered but not yet drafted.
type TournamentPlayer struct {
	TournamentID uuid.UUID
	PlayerID     uuid.UUID
	Tier         string
	Biography    string
	Hdcp         float32
	FirstName    string
	LastName     string
	Email        *string
	PhotoPath    string
	TeamID       *uuid.UUID
}

// TeamHoleScore is one side's gross score on a hole, tagged by team ID.
type TeamHoleScore struct {
	TeamID  uuid.UUID
	Strokes int32
}

// PlayerRecord is a player's win/loss/tie tally across finished matches, derived
// on read from match_results.
type PlayerRecord struct {
	Wins   int32
	Losses int32
	Ties   int32
}

// StoredResult is a match's materialized state, persisted to match_results and
// recomputed on each score write. LeaderTeamID is the current leader (nil = all
// square); the winner is LeaderTeamID once Finished. Lead and HolesRemaining give
// the margin (e.g. a "3 & 2" finish is Lead 3, HolesRemaining 2).
type StoredResult struct {
	Finished       bool
	LeaderTeamID   *uuid.UUID
	Lead           int
	HolesRemaining int
}

// HoleResult is the match-play state after a scored hole. It refers to the two
// sides by team ID — color ("Red"/"Blue") is a display attribute of the team, not
// scoring state. LeaderTeamID identifies who is ahead (nil = all square); Lead is
// the margin in holes (>= 0). Decided means the lead exceeds the holes remaining,
// so the match is closed out at this hole. Rendering this as text ("AS"/"2 UP"/
// "3 & 2") is the frontend's concern.
type HoleResult struct {
	HoleNumber     int32
	TeamScores     []TeamHoleScore // the two teams, in the order passed to ComputeMatchProgress
	LeaderTeamID   *uuid.UUID
	Lead           int
	HolesRemaining int
	Decided        bool
}

// TeamData represents a team's summary for a tournament
type TeamData struct {
	ID      uuid.UUID
	Color   string
	Captain *PlayerSummary
	Points  float64
}

// PlayerSummary is a lightweight player representation
type PlayerSummary struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     *string
}
