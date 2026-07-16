package golf

import (
	"context"
)

// logger interface for logging operations
type logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// playerDB interface defines database operations for players
type playerDB interface {
	GetPlayer(ctx context.Context, id int32) (*Player, error)
	ListPlayers(ctx context.Context) ([]Player, error)
	CreatePlayer(ctx context.Context, in CreatePlayerInput) (*Player, error)
}

// matchDB interface defines database operations for matches
type matchDB interface {
	GetMatch(ctx context.Context, id int32) (*Match, error)
	ListMatchesByTournament(ctx context.Context, tournamentID int32) ([]Match, error)
}

// participantDB interface defines database operations for match participants
type participantDB interface {
	ListMatchParticipants(ctx context.Context, matchID int32) ([]MatchParticipant, error)
}

// scoreDB interface defines database operations for scores
type scoreDB interface {
	ListScoresByMatch(ctx context.Context, matchID int32) ([]Score, error)
	// SaveScore upserts one hole score; the repo picks per-player vs team-attributable
	// storage based on Score.PlayerID being set.
	SaveScore(ctx context.Context, s Score) error
}

// teamDB interface defines database operations for teams
type teamDB interface {
	GetTeam(ctx context.Context, id int32) (*Team, error)
	ListTeamsByTournament(ctx context.Context, tournamentID int32) ([]Team, error)
}

// teamMemberDB interface defines database operations for team members
type teamMemberDB interface {
	// GetTeamCaptain returns the captain of a team (via teams.captain_id), or nil.
	GetTeamCaptain(ctx context.Context, teamID int32) (*Player, error)
}

// teeColorDB interface defines database operations for tenant-level tee colors
type teeColorDB interface {
	CreateTeeColor(ctx context.Context, in CreateTeeColorInput) (*TeeColor, error)
	ListTeeColors(ctx context.Context) ([]TeeColor, error)
}

// courseDB interface defines database operations for courses
type courseDB interface {
	CreateCourse(ctx context.Context, in CreateCourseInput) (*Course, error)
	GetCourse(ctx context.Context, id int32) (*Course, error)
	ListCourses(ctx context.Context) ([]Course, error)
}

// teeSetDB interface defines database operations for tee sets (with their holes)
type teeSetDB interface {
	CreateTeeSet(ctx context.Context, in CreateTeeSetInput) (*TeeSetWithHoles, error)
}

// formatDB interface defines read operations for global match formats
type formatDB interface {
	ListMatchFormats(ctx context.Context) ([]MatchFormat, error)
}

// tournamentDB interface defines database operations for tournaments
type tournamentDB interface {
	GetTournament(ctx context.Context, id int32) (*Tournament, error)
	ListTournaments(ctx context.Context) ([]Tournament, error)
	// CreateTournamentWithTeams inserts the tournament and one team per color in a
	// single transaction, upholding the invariant that a tournament always has its
	// full set of sides.
	CreateTournamentWithTeams(ctx context.Context, in CreateTournamentInput, teamColors []string) (*Tournament, error)
}

// resultDB reads/writes the materialized match_results and the aggregates derived
// from it (team points, tournament-finished, player records).
type resultDB interface {
	UpsertMatchResult(ctx context.Context, matchID, tournamentID int32, r StoredResult) error
	GetMatchResult(ctx context.Context, matchID int32) (*StoredResult, error)
	ListTeamPoints(ctx context.Context, tournamentID int32) (map[int32]float64, error)
	IsTournamentFinished(ctx context.Context, tournamentID int32) (bool, error)
	GetPlayerRecord(ctx context.Context, playerID int32) (PlayerRecord, error)
}
