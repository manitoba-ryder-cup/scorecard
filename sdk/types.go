package sdk

import "github.com/google/uuid"

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// ErrorResponse is the body returned for any HTTP error status.
type ErrorResponse struct {
	Error string `json:"error"`
}

// APIError is returned by SDK clients when the server responds with an error.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string { return e.Message }

// Player is a golfer's public profile. tenant_id is intentionally omitted — it is
// an internal multi-tenancy detail clients never need.
type Player struct {
	ID        int32      `json:"id"`
	UserID    *uuid.UUID `json:"user_id"` // heimdall account link; null for roster-only players
	Email     *string    `json:"email"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	PhotoPath string     `json:"photo_path"`
}

// PlayerRecord is a player's win/loss/tie tally across finished matches, derived on
// read from match_results (never stored — the old app's stale columns are gone).
type PlayerRecord struct {
	Wins   int32 `json:"wins"`
	Losses int32 `json:"losses"`
	Ties   int32 `json:"ties"`
}

// PlayerProfile is the player detail response: the base player plus the derived
// record. The list endpoint returns bare Players (no per-player record query); the
// detail endpoint pays for the extra derivation. Player is embedded, so its fields
// stay at the top level of the JSON alongside "record".
type PlayerProfile struct {
	Player
	Record PlayerRecord `json:"record"`
}

// TeeColor is a tenant-level tee marker (e.g. White, Blue), shared across courses.
type TeeColor struct {
	ID    int32  `json:"id"`
	Color string `json:"color"`
}

// CreateTeeColorRequest is the body for POST /v1/tee-colors.
type CreateTeeColorRequest struct {
	Color string `json:"color"`
}

// Course is a golf course (venue). Its tee sets and holes are added separately.
type Course struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// CreateCourseRequest is the body for POST /v1/courses.
type CreateCourseRequest struct {
	Name string `json:"name"`
}

// MatchFormat is a code-defined scoring format (e.g. Singles, Fourball). Global,
// seeded reference data — read-only over the API.
type MatchFormat struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// Hole is one hole's setup for a tee: par, stroke index (hdcp), and yardage.
type Hole struct {
	Number int32 `json:"number"`
	Par    int32 `json:"par"`
	Hdcp   int32 `json:"hdcp"`
	Yards  int32 `json:"yards"`
}

// TeeSet is a course's playable configuration for one tee color: rating/slope plus
// its 18 holes.
type TeeSet struct {
	CourseID   int32   `json:"course_id"`
	TeeColorID int32   `json:"tee_color_id"`
	Slope      int32   `json:"slope"`
	Rating     float64 `json:"rating"`
	Holes      []Hole  `json:"holes"`
}

// CreateTeeSetRequest is the body for POST /v1/courses/{id}/tees. The course comes
// from the path; tee_color_id references an existing tee color. Exactly 18 holes are
// required, with unique numbers (1-18) and unique stroke indexes (hdcp, 1-18).
type CreateTeeSetRequest struct {
	TeeColorID int32   `json:"tee_color_id"`
	Slope      int32   `json:"slope"`
	Rating     float64 `json:"rating"`
	Holes      []Hole  `json:"holes"`
}

// CreatePlayerRequest is the body for POST /v1/players. Email and user_id are
// optional (roster-only players have neither); the photo is uploaded separately.
type CreatePlayerRequest struct {
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Email     *string    `json:"email"`
	UserID    *uuid.UUID `json:"user_id"`
}

// PlayerSummary is a lightweight player reference (e.g. a team captain).
type PlayerSummary struct {
	ID        int32   `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     *string `json:"email"`
}

// Tournament is a tournament event. Dates are ISO-8601 (YYYY-MM-DD).
type Tournament struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Location  string `json:"location"`
}

// CreateTournamentRequest is the body for POST /v1/tournaments. Dates are YYYY-MM-DD.
type CreateTournamentRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Location  string `json:"location"`
}

// TournamentPlayer is a player's per-tournament attributes (their tournament entry),
// returned from enter/update. Team assignment is separate (see the draft).
type TournamentPlayer struct {
	TournamentID int32   `json:"tournament_id"`
	PlayerID     int32   `json:"player_id"`
	Tier         string  `json:"tier"`
	Biography    string  `json:"biography"`
	Hdcp         float32 `json:"hdcp"`
}

// TournamentPlayerDetail is a tournament entry joined with the player's identity, for
// roster listings.
type TournamentPlayerDetail struct {
	TournamentPlayer
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     *string `json:"email"`
	PhotoPath string  `json:"photo_path"`
}

// EnterTournamentPlayerRequest is the body for POST /v1/tournaments/{id}/players. The
// tournament comes from the path; player_id references an existing player. Attributes
// default sensibly if omitted (tier "white", empty bio, hdcp 0).
type EnterTournamentPlayerRequest struct {
	PlayerID  int32   `json:"player_id"`
	Tier      string  `json:"tier"`
	Biography string  `json:"biography"`
	Hdcp      float32 `json:"hdcp"`
}

// UpdateTournamentPlayerRequest is the body for PUT /v1/tournaments/{id}/players/{playerId}.
type UpdateTournamentPlayerRequest struct {
	Tier      string  `json:"tier"`
	Biography string  `json:"biography"`
	Hdcp      float32 `json:"hdcp"`
}

// TournamentTeam is one of a tournament's two sides with its captain and points.
type TournamentTeam struct {
	ID      int32          `json:"id"`
	Color   string         `json:"color"`
	Captain *PlayerSummary `json:"captain"`
	Points  float64        `json:"points"`
}

// ScoreSubmission is the request body for POST /v1/matches/{id}/scores — one hole
// score. course_id/tee_color_id are omitted: the server derives them from the match.
// player_id is null for one-ball team formats (alt shot, scramble).
type ScoreSubmission struct {
	HoleNumber int32  `json:"hole_number"`
	Strokes    int32  `json:"strokes"`
	TeamID     int32  `json:"team_id"`
	PlayerID   *int32 `json:"player_id"`
}

// TeamHoleScore is a side's gross score on a hole, identified by team_id.
type TeamHoleScore struct {
	TeamID  int32 `json:"team_id"`
	Strokes int32 `json:"strokes"`
}

// HoleStatus is the match-play state after a scored hole. It refers to teams by
// id (colour is a team attribute); leader_team_id is null when all square. Text
// like "2 UP" / "3 & 2" is rendered by the client from this state.
type HoleStatus struct {
	HoleNumber     int32           `json:"hole_number"`
	TeamScores     []TeamHoleScore `json:"team_scores"`
	LeaderTeamID   *int32          `json:"leader_team_id"`
	Lead           int             `json:"lead"`
	HolesRemaining int             `json:"holes_remaining"`
	Decided        bool            `json:"decided"`
}

// MatchWinnerResponse reports a match's outcome by team id (null = tie/undecided).
type MatchWinnerResponse struct {
	Finished     bool   `json:"finished"`
	WinnerTeamID *int32 `json:"winner_team_id"`
}

// TournamentWinnerResponse reports a tournament's winning side by team id.
type TournamentWinnerResponse struct {
	Finished     bool   `json:"finished"`
	WinnerTeamID *int32 `json:"winner_team_id"`
}

// FinishedResponse reports whether a match or tournament is complete.
type FinishedResponse struct {
	Finished bool `json:"finished"`
}
