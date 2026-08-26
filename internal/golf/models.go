package golf

import (
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// No tenant_id here: tenancy is an RLS concern the repositories carry in context.

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
	// Phase is derived on read from the cup's match outcomes, the way a Player's record
	// is — it is not stored, so a caller that builds a Tournament outside the service is
	// responsible for setting it.
	Phase sdk.TournamentPhase
}

// Match represents an individual golf match
type Match struct {
	ID            uuid.UUID
	TournamentID  uuid.UUID
	CourseID      uuid.UUID
	TeeColorID    uuid.UUID
	MatchFormatID uuid.UUID
	TeeTime       time.Time
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
	// Whether the format records a stroke for each player or one for the side, which is what
	// decides the shape of a hole's scores.
	ScoresPerPlayer bool
	CourseName      string
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

// Course represents a golf course.
type Course struct {
	ID   uuid.UUID
	Name string
	// TimeZone is where the course is, as an IANA name. A tee time is entered as the
	// wall clock the tee sheet says and converted to an instant against this, so an away
	// round is typed as its own local 8am. Display is the viewer's concern.
	TimeZone string
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
	ID             uuid.UUID
	Name           string
	PlayersPerSide int32
	// False is one ball for the side, not an absence of scores.
	ScoresPerPlayer bool
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

// FormatRecord is a player's W-L-T in one match format.
type FormatRecord struct {
	FormatName string
	Record     PlayerRecord
}

// PairRecord is a player's W-L-T alongside or against one other player. Matches is the
// repeat-pairing signal on its own — a captain reuses partnerships, and the count says
// whether that has been working.
type PairRecord struct {
	PlayerID  uuid.UUID
	FirstName string
	LastName  string
	Matches   int
	Record    PlayerRecord
}

// PlayerStats is a career broken down the ways a captain and a player each want it: what
// they play well, who they play well with, and who they play well against. Points is the
// cup's own currency — a win is 1, a half is ½ — and is a rate per cup rather than a total
// so that a veteran and a newcomer compare.
type PlayerStats struct {
	ByFormat   []FormatRecord
	Teammates  []PairRecord
	Opponents  []PairRecord
	Points     float64
	CupsPlayed int
	// How they fare when a match goes the distance against when it is closed out early.
	// A halved match can only land in LastHole, since a half requires playing the 18th.
	LastHole     PlayerRecord
	DecidedEarly PlayerRecord
	// The heaviest result each way. Nil for a player who has never won, or never lost.
	BestWin      *NotableMatch
	HeaviestLoss *NotableMatch
}

// PlayerStatsRows is everything PlayerStats reads, gathered in one call so the aggregates
// share a transaction. Fetched separately each paid its own BEGIN / SET LOCAL / COMMIT, and
// against a database in another region those round trips were nearly the whole response
// time — the queries themselves run in well under a millisecond. Rows only: what they mean
// stays in PlayerStats.
type PlayerStatsRows struct {
	ByFormat     []FormatRecord
	Teammates    []PairRecord
	Opponents    []PairRecord
	LastHole     PlayerRecord
	DecidedEarly PlayerRecord
	BestWin      *NotableMatch
	HeaviestLoss *NotableMatch
	History      []PlayerTournamentHistory
}

// NotableMatch is one match worth naming — the margin, who was across it, and the cup it
// happened in. Lead and HolesRemaining are left as they are rather than rendered into
// "9 & 7" here, so the margin reads the same as every other match in the app.
type NotableMatch struct {
	Year           string
	Lead           int32
	HolesRemaining int32
	Opponents      string
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
	Result           sdk.Result
	Record           PlayerRecord
	Tier             string
	Biography        string
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
	StoredResult // the closed-out state, embedded so there is one shape for it
	MatchID      uuid.UUID
	FormatName   string
	// Whether the format records a stroke for each player or one for the side.
	ScoresPerPlayer bool
	CourseName      string
	TeeTime         time.Time
	Sides           []MatchSide
	HoleResults     []*uuid.UUID
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

// MatchOutcome is a match's standing: whether anyone has scored it, whether it is
// complete, and the winning team (nil while undecided, or when the match was halved).
type MatchOutcome struct {
	Started      bool
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

// HoleResult is the match-play state after a scored hole, by team ID — colour is a display
// attribute, not scoring state. LeaderTeamID is nil when all square, Lead is the margin in
// holes, and Decided means the lead exceeds the holes remaining. Rendering it as "3 & 2" is
// the frontend's concern.
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
