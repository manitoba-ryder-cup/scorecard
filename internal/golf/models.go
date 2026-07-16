package golf

import (
	"time"

	"github.com/google/uuid"
)

// The domain layer holds no tenant_id — tenancy is a persistence/RLS concern
// carried in context by the repositories, never a domain field. Dates are plain
// time.Time; the repos map to/from the database driver's types.

// Player represents a golfer's stable identity. Per-tournament attributes (tier,
// biography, handicap) live on TeamMember; win/loss records are derived from
// match_results.
type Player struct {
	ID        int32
	UserID    *uuid.UUID // heimdall account link; nil for roster-only players
	Email     *string
	FirstName string
	LastName  string
	PhotoPath string
}

// Team represents one of a tournament's two sides.
type Team struct {
	ID           int32
	TournamentID int32
	Color        string
	CaptainID    *int32
}

// Tournament represents a golf tournament event
type Tournament struct {
	ID        int32
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Location  string
}

// Match represents an individual golf match
type Match struct {
	ID            int32
	TournamentID  int32
	CourseID      int32
	TeeColorID    int32
	MatchFormatID int32
	TeeTime       *time.Time
	Handicapped   bool
}

// MatchParticipant links a player (on a team) to a match.
type MatchParticipant struct {
	TournamentID int32
	MatchID      int32
	PlayerID     int32
	TeamID       int32
}

// Score is a hole score attributed to a side (TeamID) and, for per-player formats,
// to a player. PlayerID is nil for one-ball team scores (alt shot, scramble).
type Score struct {
	ID         int32
	MatchID    int32
	TeamID     int32
	PlayerID   *int32
	CourseID   int32
	TeeColorID int32
	HoleNumber int32
	Strokes    int32
}

// Course represents a golf course
type Course struct {
	ID   int32
	Name string
}

// Hole represents a hole on a golf course with specific tee
type Hole struct {
	CourseID   int32
	TeeColorID int32
	Number     int32
	Par        int32
	Hdcp       int32
	Yards      int32
}

// TeeColor represents tee marker colors
type TeeColor struct {
	ID    int32
	Color string
}

// TeeSet represents course rating and slope for a specific tee
type TeeSet struct {
	CourseID   int32
	TeeColorID int32
	Slope      int32
	Rating     float64
}

// MatchFormat represents the type of match
type MatchFormat struct {
	ID   int32
	Name string
}

// TeamMember is a player's membership on a team for a tournament, plus the
// player's per-tournament attributes.
type TeamMember struct {
	TeamID       int32
	PlayerID     int32
	TournamentID int32
	Tier         string
	Biography    string
	Hdcp         float32
}

// TeamHoleScore is one side's gross score on a hole, tagged by team ID.
type TeamHoleScore struct {
	TeamID  int32
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
	LeaderTeamID   *int32
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
	LeaderTeamID   *int32
	Lead           int
	HolesRemaining int
	Decided        bool
}

// TeamData represents a team's summary for a tournament
type TeamData struct {
	ID      int32
	Color   string
	Captain *PlayerSummary
	Points  float64
}

// PlayerSummary is a lightweight player representation
type PlayerSummary struct {
	ID        int32
	FirstName string
	LastName  string
	Email     *string
}
