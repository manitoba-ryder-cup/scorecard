package golf

import (
	"time"

	"github.com/google/uuid"
)

// The domain layer holds no tenant_id — tenancy is a persistence/RLS concern
// carried in context by the repositories, never a domain field. Entity IDs are all
// uuid.UUID (matching the schema); non-ID integers (hole number, par, strokes) stay
// int. Dates are plain time.Time; the repos map to/from the database driver's types.

// Player is a golfer's public profile: stable identity plus their all-time record and
// cups won, both derived on read (0 for a new player). Per-tournament attributes (tier,
// biography, handicap) live on TournamentPlayer.
type Player struct {
	ID     uuid.UUID
	UserID *uuid.UUID // heimdall account link; nil for roster-only players
	// Email is never serialized; it is the seed CLI's key for matching a returning
	// player, so it stays on the domain model but off the wire.
	Email     *string
	FirstName string
	LastName  string
	PhotoPath string
	Record    PlayerRecord
	CupsWon   int // finished tournaments the player's team won outright
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

// MatchParticipantPlayer is a match participant enriched with the player's name.
type MatchParticipantPlayer struct {
	MatchID   uuid.UUID
	TeamID    uuid.UUID
	PlayerID  uuid.UUID
	FirstName string
	LastName  string
}

// MatchDetail is a match plus its resolved format and course names.
type MatchDetail struct {
	Match
	FormatName string
	CourseName string
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

// CourseTeeSet is a course's tee set with its colour name resolved — the shape a
// match-setup picker needs to offer valid, labelled (course, tee) options.
type CourseTeeSet struct {
	CourseID   uuid.UUID
	TeeColorID uuid.UUID
	Color      string
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
// Record and CupsWon are all-time, derived from match_results; only the roster listing
// populates them, other reads leave them zero.
type TournamentPlayer struct {
	TournamentID uuid.UUID
	PlayerID     uuid.UUID
	Tier         string
	Biography    string
	Hdcp         float32
	FirstName    string
	LastName     string
	PhotoPath    string
	TeamID       *uuid.UUID
	Record       PlayerRecord
	CupsWon      int
}

// PlayerTournamentHistory is one event in a player's history: their team that year
// (by its captain), the outcome, and their W-L-T in that tournament. TeamID feeds the
// Result derivation and is not put on the wire; Result is filled by the service.
type PlayerTournamentHistory struct {
	TournamentID     uuid.UUID
	Name             string
	Location         string
	StartDate        time.Time
	EndDate          time.Time
	CaptainFirstName string
	CaptainLastName  string
	TeamID           *uuid.UUID // nil when entered but never drafted
	Result           string
	Record           PlayerRecord
}

// MatchSidePlayer is a player on one side of a match.
type MatchSidePlayer struct {
	PlayerID  uuid.UUID
	FirstName string
	LastName  string
}

// MatchSide is one team's lineup in a match, by id.
type MatchSide struct {
	TeamID  uuid.UUID
	Players []MatchSidePlayer
}

// MatchResult is a match's outcome for the tournament results view. HoleResults holds,
// per played hole in order, the winning team's id (nil = halved); its length is the
// holes played. The closed-out state (Finished/WinnerTeamID/Lead/HolesRemaining) is
// the same StoredResult the scoring engine persists.
type MatchResult struct {
	MatchID      uuid.UUID
	FormatName   string
	CourseName   string
	TeeTime      *time.Time
	Finished     bool
	WinnerTeamID *uuid.UUID
	// LeaderTeamID is who is ahead right now (nil = all square), set whether or not the
	// match has finished. A live leaderboard needs it; without it every client has to
	// re-derive the leader by counting HoleResults.
	LeaderTeamID   *uuid.UUID
	Lead           int
	HolesRemaining int
	Sides          []MatchSide
	HoleResults    []*uuid.UUID
}

// TeamHoleScore is one side's gross score on a hole, tagged by team ID. Strokes alone
// can't be unwound — it's the best ball in fourball — so PlayerScores keeps the
// individual scores behind it, ordered by player ID. Nil for a one-ball format, where
// the score is recorded against the team.
type TeamHoleScore struct {
	TeamID       uuid.UUID
	Strokes      int32
	PlayerScores []PlayerHoleScore
}

// PlayerHoleScore is one player's strokes on a hole.
type PlayerHoleScore struct {
	PlayerID uuid.UUID
	Strokes  int32
}

// PlayerRecord is a player's win/loss/tie tally across finished matches, derived
// on read from match_results.
type PlayerRecord struct {
	Wins   int32
	Losses int32
	Ties   int32
}

// MatchOutcome is a match's standing: whether it is complete, and the winning team
// (nil while undecided, or when the match was halved).
type MatchOutcome struct {
	Finished     bool
	WinnerTeamID *uuid.UUID
}

// TournamentOutcome is a tournament's standing: whether every match is final, and the
// winning team (nil when unfinished or tied).
type TournamentOutcome struct {
	Finished     bool
	WinnerTeamID *uuid.UUID
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

// Winner is the leader once the match is decided, nil while it is live or halved.
// The one place "the leader is the winner once it's finished" is written down.
func (r StoredResult) Winner() *uuid.UUID {
	if !r.Finished {
		return nil
	}
	return r.LeaderTeamID
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

// TeamWithCaptain is a team plus its resolved captain (nil if unset). Returned by the
// team listing so the tournament summary needs no per-team captain lookup.
type TeamWithCaptain struct {
	Team
	Captain *PlayerSummary
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
}
