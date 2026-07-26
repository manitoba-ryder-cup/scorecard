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

// Player is a golfer's public profile. tenant_id and email are intentionally omitted:
// reads are public, so an address would be published to anonymous spectators. Email is
// write-only, kept server-side as the seed CLI's key for recognising a returning player.
type Player struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id"` // heimdall account link; null for roster-only players
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

// PlayerProfile is a player with their derived record and cups won, returned by both
// the detail and list endpoints. Player is embedded, so its fields stay at the top level
// of the JSON alongside "record".
type PlayerProfile struct {
	Player
	Record  PlayerRecord `json:"record"`
	CupsWon int          `json:"cups_won"` // finished tournaments the player's team won
}

// TeeColor is a tenant-level tee marker (e.g. White, Blue), shared across courses.
type TeeColor struct {
	ID    uuid.UUID `json:"id"`
	Color string    `json:"color"`
}

// CreateTeeColorRequest is the body for POST /v1/tee-colors.
type CreateTeeColorRequest struct {
	Color string `json:"color"`
}

// Course is a golf course (venue). Its tee sets and holes are added separately.
type Course struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// CreateCourseRequest is the body for POST /v1/courses.
type CreateCourseRequest struct {
	Name string `json:"name"`
}

// MatchFormat is a code-defined scoring format (e.g. Singles, Fourball). Global,
// seeded reference data — read-only over the API.
type MatchFormat struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
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
	CourseID   uuid.UUID `json:"course_id"`
	TeeColorID uuid.UUID `json:"tee_color_id"`
	Slope      int32     `json:"slope"`
	Rating     float64   `json:"rating"`
	Holes      []Hole    `json:"holes"`
}

// TeeSetSummary is a course's tee set with its colour name resolved and no holes — the
// labelled (course, tee) option a match-setup picker offers.
type TeeSetSummary struct {
	CourseID   uuid.UUID `json:"course_id"`
	TeeColorID uuid.UUID `json:"tee_color_id"`
	Color      string    `json:"color"`
	Slope      int32     `json:"slope"`
	Rating     float64   `json:"rating"`
}

// CreateTeeSetRequest is the body for POST /v1/courses/{id}/tees. The course comes
// from the path; tee_color_id references an existing tee color. Exactly 18 holes are
// required, with unique numbers (1-18) and unique stroke indexes (hdcp, 1-18).
type CreateTeeSetRequest struct {
	TeeColorID uuid.UUID `json:"tee_color_id"`
	Slope      int32     `json:"slope"`
	Rating     float64   `json:"rating"`
	Holes      []Hole    `json:"holes"`
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
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

// Tournament is a tournament event. Dates are ISO-8601 (YYYY-MM-DD).
type Tournament struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Location  string    `json:"location"`
}

// CreateTournamentRequest is the body for POST /v1/tournaments. Dates are YYYY-MM-DD.
type CreateTournamentRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Location  string `json:"location"`
}

// TournamentPlayer is a player entered in a tournament: the per-tournament attributes
// (tier, biography, handicap) set independently of the draft, the player's identity,
// and their team assignment. team_id is null when entered but not yet drafted. record
// (all-time W-L-T) and cups_won are populated by the roster listing only.
type TournamentPlayer struct {
	TournamentID uuid.UUID    `json:"tournament_id"`
	PlayerID     uuid.UUID    `json:"player_id"`
	Tier         string       `json:"tier"`
	Biography    string       `json:"biography"`
	Hdcp         float32      `json:"hdcp"`
	FirstName    string       `json:"first_name"`
	LastName     string       `json:"last_name"`
	PhotoPath    string       `json:"photo_path"`
	TeamID       *uuid.UUID   `json:"team_id"`
	Record       PlayerRecord `json:"record"`
	CupsWon      int          `json:"cups_won"`
}

// PlayerTournamentHistory is one event in a player's history: the event metadata, their
// team that year (identified by its captain), the outcome, and their W-L-T in that
// tournament. result is one of the Result* constants.
type PlayerTournamentHistory struct {
	TournamentID     uuid.UUID    `json:"tournament_id"`
	Name             string       `json:"name"`
	Location         string       `json:"location"`
	StartDate        string       `json:"start_date"`
	EndDate          string       `json:"end_date"`
	CaptainFirstName string       `json:"captain_first_name"`
	CaptainLastName  string       `json:"captain_last_name"`
	Result           string       `json:"result"`
	Record           PlayerRecord `json:"record"`
}

// EnterTournamentPlayerRequest is the body for POST /v1/tournaments/{id}/players. The
// tournament comes from the path; player_id references an existing player. Attributes
// default sensibly if omitted (tier "white", empty bio, hdcp 0).
type EnterTournamentPlayerRequest struct {
	PlayerID  uuid.UUID `json:"player_id"`
	Tier      string    `json:"tier"`
	Biography string    `json:"biography"`
	Hdcp      float32   `json:"hdcp"`
}

// UpdateTournamentPlayerRequest is the body for PUT /v1/tournaments/{id}/players/{playerId}.
type UpdateTournamentPlayerRequest struct {
	Tier      string  `json:"tier"`
	Biography string  `json:"biography"`
	Hdcp      float32 `json:"hdcp"`
}

// TeamMember is the draft outcome: a player assigned to a team for a tournament.
type TeamMember struct {
	TeamID       uuid.UUID `json:"team_id"`
	PlayerID     uuid.UUID `json:"player_id"`
	TournamentID uuid.UUID `json:"tournament_id"`
}

// DraftPlayerRequest is the body for POST /v1/teams/{id}/members. The team (and its
// tournament) come from the path; the player must already be entered in the tournament.
type DraftPlayerRequest struct {
	PlayerID uuid.UUID `json:"player_id"`
}

// SetTeamCaptainRequest is the body for PUT /v1/teams/{id}/captain. The captain must be
// an existing player (typically one drafted onto the team).
type SetTeamCaptainRequest struct {
	CaptainID uuid.UUID `json:"captain_id"`
}

// TournamentTeam is one of a tournament's two sides with its captain and points.
type TournamentTeam struct {
	ID      uuid.UUID      `json:"id"`
	Color   string         `json:"color"`
	Captain *PlayerSummary `json:"captain"`
	Points  float64        `json:"points"`
}

// Match is a scheduled match within a tournament. tee_time is RFC3339 (null if
// unscheduled); handicapped toggles net scoring for this match.
type Match struct {
	ID            uuid.UUID `json:"id"`
	TournamentID  uuid.UUID `json:"tournament_id"`
	CourseID      uuid.UUID `json:"course_id"`
	TeeColorID    uuid.UUID `json:"tee_color_id"`
	MatchFormatID uuid.UUID `json:"match_format_id"`
	TeeTime       *string   `json:"tee_time"`
	Handicapped   bool      `json:"handicapped"`
}

// CreateMatchRequest is the body for POST /v1/tournaments/{id}/matches. The tournament
// comes from the path; course_id/tee_color_id must reference a configured tee set, and
// match_format_id one of the seeded formats. tee_time (RFC3339) and handicapped are
// optional.
type CreateMatchRequest struct {
	CourseID      uuid.UUID `json:"course_id"`
	TeeColorID    uuid.UUID `json:"tee_color_id"`
	MatchFormatID uuid.UUID `json:"match_format_id"`
	TeeTime       *string   `json:"tee_time"`
	Handicapped   bool      `json:"handicapped"`
}

// MatchParticipant is a player (on a team) taking part in a match.
type MatchParticipant struct {
	TournamentID uuid.UUID `json:"tournament_id"`
	MatchID      uuid.UUID `json:"match_id"`
	PlayerID     uuid.UUID `json:"player_id"`
	TeamID       uuid.UUID `json:"team_id"`
}

// AddParticipantRequest is the body for POST /v1/matches/{id}/participants. The match
// (and its tournament) come from the path; the player must be drafted onto team_id.
type AddParticipantRequest struct {
	PlayerID uuid.UUID `json:"player_id"`
	TeamID   uuid.UUID `json:"team_id"`
}

// ScoreSubmission is the request body for POST /v1/matches/{id}/scores — one hole
// score. course_id/tee_color_id are omitted: the server derives them from the match.
// player_id is null for one-ball team formats (alt shot, scramble).
type ScoreSubmission struct {
	HoleNumber int32      `json:"hole_number"`
	Strokes    int32      `json:"strokes"`
	TeamID     uuid.UUID  `json:"team_id"`
	PlayerID   *uuid.UUID `json:"player_id"`
}

// TeamHoleScore is a side's gross score on a hole, identified by team_id. strokes is the
// best ball in fourball, so player_scores carries what each player actually shot, ordered
// by player_id. Empty for a one-ball format, where the score belongs to the team.
type TeamHoleScore struct {
	TeamID       uuid.UUID         `json:"team_id"`
	Strokes      int32             `json:"strokes"`
	PlayerScores []PlayerHoleScore `json:"player_scores"`
}

// PlayerHoleScore is one player's strokes on a hole.
type PlayerHoleScore struct {
	PlayerID uuid.UUID `json:"player_id"`
	Strokes  int32     `json:"strokes"`
}

// HoleStatus is the match-play state after a scored hole. It refers to teams by
// id (colour is a team attribute); leader_team_id is null when all square. decided
// marks the hole the match ended on; it ended early when holes_remaining > 0. Text
// like "2 UP" / "3 & 2" is rendered by the client from this state.
type HoleStatus struct {
	HoleNumber     int32           `json:"hole_number"`
	TeamScores     []TeamHoleScore `json:"team_scores"`
	LeaderTeamID   *uuid.UUID      `json:"leader_team_id"`
	Lead           int             `json:"lead"`
	HolesRemaining int             `json:"holes_remaining"`
	Decided        bool            `json:"decided"`
}

// MatchPlayer is a player on one side of a match.
type MatchPlayer struct {
	PlayerID  uuid.UUID `json:"player_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

// MatchSide is one team's lineup in a match, by id. Colour is resolved by the client
// from the tournament's teams.
type MatchSide struct {
	TeamID  uuid.UUID     `json:"team_id"`
	Players []MatchPlayer `json:"players"`
}

// MatchResult is a match's outcome for the tournament results view. hole_results holds,
// per played hole in order, the winning team's id (null = halved); its length is the
// number of holes played. winner_team_id is null unless finished, while leader_team_id
// reports who is ahead at any point (null = all square) so a live leaderboard needs no
// client-side derivation. tee_time is RFC3339 (null if unscheduled).
type MatchResult struct {
	MatchStatus              // finished/winner/leader/lead/holes_remaining, flattened into this object
	MatchID     uuid.UUID    `json:"match_id"`
	FormatName  string       `json:"format_name"`
	Sides       []MatchSide  `json:"sides"`
	HoleResults []*uuid.UUID `json:"hole_results"`
	TeeTime     *string      `json:"tee_time"`
	CourseName  string       `json:"course_name"`
}

// MatchStatus is a match's outcome state: the one shape for it, returned by the match
// status/winner reads, by a score write (so the client learns the new state from the
// write instead of re-deriving the close-out rule), and embedded in MatchResult.
// winner_team_id is the leader once finished, null otherwise; leader_team_id reports who
// is ahead at any point (null = all square) so a live leaderboard needs no derivation.
type MatchStatus struct {
	Finished       bool       `json:"finished"`
	WinnerTeamID   *uuid.UUID `json:"winner_team_id"`
	LeaderTeamID   *uuid.UUID `json:"leader_team_id"`
	Lead           int        `json:"lead"`
	HolesRemaining int        `json:"holes_remaining"`
}

// WinnerResponse reports a winning team by id (null = tie/undecided) for a tournament.
// A match answers with the richer MatchStatus.
type WinnerResponse struct {
	Finished     bool       `json:"finished"`
	WinnerTeamID *uuid.UUID `json:"winner_team_id"`
}

// FinishedResponse reports whether a tournament is complete.
type FinishedResponse struct {
	Finished bool `json:"finished"`
}
