package golf

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Team name constants
const (
	TeamRed  = "Red"
	TeamBlue = "Blue"
	TeamTied = "Tied"
)

// Player represents a golfer's stable identity and career totals. Per-tournament
// attributes (tier, biography, handicap) live on TeamMember.
type Player struct {
	ID        int32
	TenantID  uuid.UUID
	UserID    *uuid.UUID // heimdall account link; nil for roster-only players
	Email     *string
	FirstName string
	LastName  string
	PhotoPath string
	Cups      int32
	Wins      int32
	Ties      int32
	Losses    int32
}

// Team represents one of a tournament's two sides.
type Team struct {
	ID           int32
	TenantID     uuid.UUID
	TournamentID int32
	Color        string
	CaptainID    *int32
}

// Tournament represents a golf tournament event
type Tournament struct {
	ID        int32
	TenantID  uuid.UUID
	Name      string
	StartDate pgtype.Date
	EndDate   pgtype.Date
	Location  string
}

// Match represents an individual golf match
type Match struct {
	ID            int32
	TournamentID  int32
	CourseID      int32
	TeeColorID    int32
	MatchFormatID int32
	TenantID      uuid.UUID
	TeeTime       pgtype.Timestamp
	Handicapped   bool
}

// MatchParticipant links a player (on a team) to a match.
type MatchParticipant struct {
	TournamentID int32
	MatchID      int32
	PlayerID     int32
	TeamID       int32
	TenantID     uuid.UUID
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
	TenantID   uuid.UUID
	Strokes    int32
}

// Course represents a golf course
type Course struct {
	ID       int32
	TenantID uuid.UUID
	Name     string
}

// Hole represents a hole on a golf course with specific tee
type Hole struct {
	CourseID   int32
	TeeColorID int32
	Number     int32
	TenantID   uuid.UUID
	Par        int32
	Hdcp       int32
	Yards      int32
}

// TeeColor represents tee marker colors
type TeeColor struct {
	ID       int32
	TenantID uuid.UUID
	Color    string
}

// TeeSet represents course rating and slope for a specific tee
type TeeSet struct {
	CourseID   int32
	TeeColorID int32
	TenantID   uuid.UUID
	Slope      int32
	Rating     pgtype.Numeric
}

// MatchFormat represents the type of match
type MatchFormat struct {
	ID       int32
	TenantID uuid.UUID
	Name     string
}

// TeamMember is a player's membership on a team for a tournament, plus the
// player's per-tournament attributes.
type TeamMember struct {
	TeamID       int32
	PlayerID     int32
	TournamentID int32
	TenantID     uuid.UUID
	Tier         string
	Biography    string
	Hdcp         float32
}

// TeamHoleScore is one side's gross score on a hole, tagged by team ID.
type TeamHoleScore struct {
	TeamID  int32
	Strokes int32
}

// MatchResult is the decided outcome of a match. WinnerTeamID is nil for a tie
// (all square through 18); both fields are zero-valued while the match is unfinished.
type MatchResult struct {
	Finished     bool
	WinnerTeamID *int32
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
