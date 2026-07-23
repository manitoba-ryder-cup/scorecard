package golf

import (
	"context"

	"github.com/google/uuid"
)

// playerDB interface defines database operations for players
type playerDB interface {
	GetPlayer(ctx context.Context, id uuid.UUID) (*Player, error)
	ListPlayers(ctx context.Context) ([]Player, error)
	CreatePlayer(ctx context.Context, in CreatePlayerInput) (*Player, error)
	// Result is left unset — the service derives it from the standings.
	ListPlayerTournaments(ctx context.Context, playerID uuid.UUID) ([]PlayerTournamentHistory, error)
}

// matchDB interface defines database operations for matches
type matchDB interface {
	GetMatch(ctx context.Context, id uuid.UUID) (*Match, error)
	ListMatchesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Match, error)
	ListMatchDetailsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]MatchDetail, error)
	CreateMatch(ctx context.Context, in CreateMatchInput) (*Match, error)
}

// participantDB interface defines database operations for match participants
type participantDB interface {
	ListMatchParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error)
	ListParticipantsWithPlayersByTournament(ctx context.Context, tournamentID uuid.UUID) ([]MatchParticipantPlayer, error)
	CreateMatchParticipant(ctx context.Context, tournamentID, matchID, playerID, teamID uuid.UUID) (*MatchParticipant, error)
}

// scoreDB interface defines database operations for scores
type scoreDB interface {
	ListScoresByMatch(ctx context.Context, matchID uuid.UUID) ([]Score, error)
	ListScoresByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Score, error)
	// SaveScore upserts one hole score; the repo picks per-player vs team-attributable
	// storage based on Score.PlayerID being set.
	SaveScore(ctx context.Context, s Score) error
}

// holeDB reads a tee set's holes (course setup).
type holeDB interface {
	ListHolesByTeeSet(ctx context.Context, courseID, teeColorID uuid.UUID) ([]Hole, error)
}

// teamDB interface defines database operations for teams
type teamDB interface {
	GetTeam(ctx context.Context, id uuid.UUID) (*Team, error)
	// ListTeamsByTournament returns the tournament's teams with their captains resolved.
	ListTeamsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]TeamWithCaptain, error)
	// SetTeamCaptain assigns a team's captain and returns the updated team.
	SetTeamCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*Team, error)
}

// teamMemberDB interface defines database operations for team members
type teamMemberDB interface {
	// CreateTeamMember drafts a player onto a team (the tournament is the team's).
	CreateTeamMember(ctx context.Context, teamID, playerID, tournamentID uuid.UUID) (*TeamMember, error)
}

// tournamentPlayerDB interface defines database operations for tournament entries
type tournamentPlayerDB interface {
	CreateTournamentPlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error)
	UpdateTournamentPlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error)
	ListTournamentPlayers(ctx context.Context, tournamentID uuid.UUID) ([]TournamentPlayer, error)
	ListTournamentPlayersByTeam(ctx context.Context, teamID uuid.UUID) ([]TournamentPlayer, error)
}

// teeColorDB interface defines database operations for tenant-level tee colors
type teeColorDB interface {
	CreateTeeColor(ctx context.Context, in CreateTeeColorInput) (*TeeColor, error)
	ListTeeColors(ctx context.Context) ([]TeeColor, error)
}

// courseDB interface defines database operations for courses
type courseDB interface {
	CreateCourse(ctx context.Context, in CreateCourseInput) (*Course, error)
	GetCourse(ctx context.Context, id uuid.UUID) (*Course, error)
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
	GetTournament(ctx context.Context, id uuid.UUID) (*Tournament, error)
	ListTournaments(ctx context.Context) ([]Tournament, error)
	// CreateTournamentWithTeams inserts the tournament and one team per color in a
	// single transaction, upholding the invariant that a tournament always has its
	// full set of sides.
	CreateTournamentWithTeams(ctx context.Context, in CreateTournamentInput, teamColors []string) (*Tournament, error)
}

// resultDB reads/writes the materialized match_results and the aggregates derived
// from it (team points, tournament-finished, player records).
type resultDB interface {
	UpsertMatchResult(ctx context.Context, matchID, tournamentID uuid.UUID, r StoredResult) error
	GetMatchResult(ctx context.Context, matchID uuid.UUID) (*StoredResult, error)
	ListTeamPoints(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]float64, error)
	IsTournamentFinished(ctx context.Context, tournamentID uuid.UUID) (bool, error)
	// Batched over a tournament's roster, each keyed by player id.
	ListTournamentPlayerRecords(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]PlayerRecord, error)
	ListTournamentPlayerCups(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]int, error)
}
