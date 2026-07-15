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

// Player represents a golfer with handicap and statistics
type Player struct {
	ID        int32
	TenantID  uuid.UUID
	Email     string
	FirstName string
	LastName  string
	Hdcp      float32
	PhotoPath *string
	Biography *string
	Tier      string
	Cups      int32
	Wins      int32
	Ties      int32
	Losses    int32
}

// Team represents a team in a tournament
type Team struct {
	ID       int32
	TenantID uuid.UUID
	Name     string
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

// MatchParticipant links a player to a match
type MatchParticipant struct {
	TournamentID int32
	MatchID      int32
	PlayerID     int32
	TenantID     uuid.UUID
}

// Score represents a player's score on a specific hole
type Score struct {
	MatchID    int32
	PlayerID   int32
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

// TeamMember represents a player's membership on a team for a tournament
type TeamMember struct {
	TournamentID int32
	PlayerID     int32
	TeamID       int32
	TenantID     uuid.UUID
	IsCaptain    bool
}

// MatchStatus represents the state of a match at a specific hole
type MatchStatus struct {
	MatchStatus   int    // Cumulative lead (+ = Red ahead, - = Blue ahead, 0 = All Square)
	StatusText    string // Human-readable status: "2 UP", "AS", "3 & 2", etc.
	RedTeamScore  int32  // Red team's score on this hole
	BlueTeamScore int32  // Blue team's score on this hole
}

// TeamData represents a team's summary for a tournament
type TeamData struct {
	ID      int32
	Name    string
	Captain *PlayerSummary
	Points  float64
}

// PlayerSummary is a lightweight player representation
type PlayerSummary struct {
	ID        int32
	FirstName string
	LastName  string
	Email     string
}
