package sdk

import "github.com/google/uuid"

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
// read from match_results and never stored, so it cannot go stale.
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
// time_zone is an IANA name: a tee time is entered as the wall clock the tee sheet says
// and converted against it, so an away round is typed as its own local time. Display is
// the viewer's concern, so nothing reads this back to render.
type Course struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	TimeZone string    `json:"time_zone"`
}

// CreateCourseRequest is the body for POST /v1/courses. time_zone is an IANA name;
// empty defaults to DefaultTimeZone.
type CreateCourseRequest struct {
	Name     string `json:"name"`
	TimeZone string `json:"time_zone"`
}

// MatchFormat is a scoring format (e.g. Singles, Fourball). Global, seeded reference
// data — read-only over the API. players_per_side and scores_per_player are its rules.
type MatchFormat struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	PlayersPerSide  int32     `json:"players_per_side"`
	ScoresPerPlayer bool      `json:"scores_per_player"`
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

// UpdatePlayerRequest is the body for PUT /v1/players/{id}. An omitted field keeps its
// stored value, and a body setting none is rejected rather than treated as a no-op.
//
// Changing an email means changing it in next year's setup file too, which matches a
// returning player on it. Blanking one is refused for the same reason: it would orphan
// them from every future seed.
type UpdatePlayerRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	PhotoPath *string `json:"photo_path,omitempty"`
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
	// Phase is the server's own answer to whether the cup is upcoming, being played, or
	// over. Derived from its matches, not from the dates above — a cup whose start date has
	// arrived is still upcoming until someone records a score.
	Phase TournamentPhase `json:"phase"`
}

// CreateTournamentRequest is the body for POST /v1/tournaments. Dates are YYYY-MM-DD.
type CreateTournamentRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Location  string `json:"location"`
}

// DefaultTimeZone is where the cup has always been played, used when a course is created
// without naming a zone.
const DefaultTimeZone = "America/Winnipeg"

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
	Result           Result       `json:"result"`
	Record           PlayerRecord `json:"record"`
	Tier             string       `json:"tier"`
	Biography        string       `json:"biography"`
}

// FormatRecord is a player's W-L-T in one match format.
type FormatRecord struct {
	FormatName string       `json:"format_name"`
	Record     PlayerRecord `json:"record"`
}

// PairRecord is a player's W-L-T alongside or against one other player. matches doubles as
// the repeat-pairing signal: captains reuse partnerships, and the count says whether it
// has been working.
type PairRecord struct {
	PlayerID  uuid.UUID    `json:"player_id"`
	FirstName string       `json:"first_name"`
	LastName  string       `json:"last_name"`
	Matches   int          `json:"matches"`
	Record    PlayerRecord `json:"record"`
}

// PlayerStats is the body of GET /v1/players/{id}/stats: a career split the ways a captain
// and a player each want to read it. points is the cup's own currency (a win is 1, a half
// is ½) reported next to the cups it was earned over, so the rate is the caller's to
// compute and round.
type PlayerStats struct {
	ByFormat   []FormatRecord `json:"by_format"`
	Teammates  []PairRecord   `json:"teammates"`
	Opponents  []PairRecord   `json:"opponents"`
	Points     float64        `json:"points"`
	CupsPlayed int            `json:"cups_played"`
	// How they fare when a match goes the distance against when it is closed out early.
	// A halved match can only be in last_hole, since a half requires playing the 18th.
	LastHole     PlayerRecord `json:"last_hole"`
	DecidedEarly PlayerRecord `json:"decided_early"`
	// The heaviest result each way, null for a player who has never won or never lost.
	BestWin      *NotableMatch `json:"best_win"`
	HeaviestLoss *NotableMatch `json:"heaviest_loss"`
}

// NotableMatch is one match worth naming. lead and holes_remaining are the raw margin
// rather than a rendered "9 & 7", so a caller formats it the same way it formats every
// other match.
type NotableMatch struct {
	Year           string `json:"year"`
	Lead           int32  `json:"lead"`
	HolesRemaining int32  `json:"holes_remaining"`
	Opponents      string `json:"opponents"`
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
// Omitted fields keep their stored value; a body that sets none is rejected rather than
// treated as a no-op. Partial because a biography is usually written by someone who has no
// reason to know the player's handicap, and a full replacement would zero it.
type UpdateTournamentPlayerRequest struct {
	Tier      *string  `json:"tier,omitempty"`
	Biography *string  `json:"biography,omitempty"`
	Hdcp      *float32 `json:"hdcp,omitempty"`
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

// Match is a scheduled match within a tournament. tee_time is RFC3339 and required — it
// is the instant the match's scoring window opens and closes around. handicapped is stored
// for a net-scoring mode that does not exist yet; every match is scored gross.
type Match struct {
	ID            uuid.UUID `json:"id"`
	TournamentID  uuid.UUID `json:"tournament_id"`
	CourseID      uuid.UUID `json:"course_id"`
	TeeColorID    uuid.UUID `json:"tee_color_id"`
	MatchFormatID uuid.UUID `json:"match_format_id"`
	TeeTime       string    `json:"tee_time"`
	Handicapped   bool      `json:"handicapped"`
}

// CreateMatchRequest is the body for POST /v1/tournaments/{id}/matches. The tournament
// comes from the path; course_id/tee_color_id must reference a configured tee set, and
// match_format_id one of the seeded formats. tee_time (RFC3339) is required — it is what
// the match's scoring window is measured from; handicapped is optional.
type CreateMatchRequest struct {
	CourseID      uuid.UUID `json:"course_id"`
	TeeColorID    uuid.UUID `json:"tee_color_id"`
	MatchFormatID uuid.UUID `json:"match_format_id"`
	TeeTime       string    `json:"tee_time"`
	Handicapped   bool      `json:"handicapped"`
}

// UpdateMatchRequest is the body for PUT /v1/matches/{id}. Omitted fields keep their
// stored value; a body that sets none is rejected rather than treated as a no-op.
//
// The tournament and the format are absent on purpose. Scores and participants reference the
// tournament, and the format decides how many play a side and whether a hole is recorded per
// player — changing either reinterprets a match rather than adjusting it. Delete it instead.
type UpdateMatchRequest struct {
	CourseID    *uuid.UUID `json:"course_id,omitempty"`
	TeeColorID  *uuid.UUID `json:"tee_color_id,omitempty"`
	TeeTime     *string    `json:"tee_time,omitempty"`
	Handicapped *bool      `json:"handicapped,omitempty"`
}

// MatchParticipant is a player (on a team) taking part in a match.
type MatchParticipant struct {
	TournamentID uuid.UUID `json:"tournament_id"`
	MatchID      uuid.UUID `json:"match_id"`
	PlayerID     uuid.UUID `json:"player_id"`
	TeamID       uuid.UUID `json:"team_id"`
}

// LineupPlayer is one player on one side of a match. The player must be drafted onto team_id.
type LineupPlayer struct {
	PlayerID uuid.UUID `json:"player_id"`
	TeamID   uuid.UUID `json:"team_id"`
}

// SetLineupRequest is the body for PUT /v1/matches/{id}/participants: the whole lineup, both
// sides at once. A lineup is submitted complete rather than a player at a time, so the server
// decides against the set it is being asked to write instead of the one it happens to hold.
type SetLineupRequest struct {
	Participants []LineupPlayer `json:"participants"`
}

// ScoreSubmission is the request body for POST /v1/matches/{id}/scores: every score for
// one hole, recorded together. The hole is written whole or not at all, so a dropped
// connection cannot leave one side scored and the other not. course_id/tee_color_id are
// omitted — the server derives them from the match.
type ScoreSubmission struct {
	HoleNumber int32        `json:"hole_number"`
	Scores     []ScoreEntry `json:"scores"`
}

// ScoreEntry is one competitor's score on the hole. player_id is null for one-ball team
// formats (alt shot, scramble), where the score belongs to the team rather than a player.
type ScoreEntry struct {
	TeamID   uuid.UUID  `json:"team_id"`
	PlayerID *uuid.UUID `json:"player_id"`
	Strokes  int32      `json:"strokes"`
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
// client-side derivation. tee_time is RFC3339.
type MatchResult struct {
	MatchStatus           // finished/winner/leader/lead/holes_remaining, flattened into this object
	MatchID     uuid.UUID `json:"match_id"`
	FormatName  string    `json:"format_name"`
	// How many a side the format fields. Sent so a client drawing a match with no lineup yet
	// takes the shape of the pairing from the format rather than from a list of format names.
	PlayersPerSide int32 `json:"players_per_side"`
	// Whether the format records a stroke for each player or one for the side. Sent so a client
	// reading a hole takes the grain from the format rather than from a list of format names.
	ScoresPerPlayer bool         `json:"scores_per_player"`
	Sides           []MatchSide  `json:"sides"`
	HoleResults     []*uuid.UUID `json:"hole_results"`
	TeeTime         string       `json:"tee_time"`
	CourseName      string       `json:"course_name"`
	// The scoring window, RFC3339. Both bounds are sent so the client gates its UI on the
	// server's rule rather than a second copy of it, and sent as instants rather than a
	// yes/no so they stay correct on a page left open across the boundary.
	ScoringOpensAt  string `json:"scoring_opens_at"`
	ScoringClosesAt string `json:"scoring_closes_at"`
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
