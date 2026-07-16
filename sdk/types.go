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

// Team is a tournament side in its raw form (no derived points), returned when a
// team is created. See TournamentTeam for the standings view with captain + points.
type Team struct {
	ID           int32  `json:"id"`
	TournamentID int32  `json:"tournament_id"`
	Color        string `json:"color"`
	CaptainID    *int32 `json:"captain_id"`
}

// CreateTeamRequest is the body for POST /v1/tournaments/{id}/teams. The tournament
// comes from the path; color must be Red or Blue; captain_id is optional.
type CreateTeamRequest struct {
	Color     string `json:"color"`
	CaptainID *int32 `json:"captain_id"`
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
