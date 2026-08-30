package golf

import (
	"context"

	"github.com/google/uuid"
)

type playerDB interface {
	GetPlayer(ctx context.Context, id uuid.UUID) (*Player, error)
	ListPlayers(ctx context.Context) ([]Player, error)
	CreatePlayer(ctx context.Context, in CreatePlayerInput) (*Player, error)
	UpdatePlayer(ctx context.Context, in UpdatePlayerInput) (*Player, error)
	// Result is left unset — the service derives it from the standings.
	ListPlayerTournaments(ctx context.Context, playerID uuid.UUID) ([]PlayerTournamentHistory, error)
	// One call rather than one per aggregate, so they share a transaction — see
	// PlayerStatsRows for why that is worth a wider port.
	PlayerStatsRows(ctx context.Context, playerID uuid.UUID) (*PlayerStatsRows, error)
}

type matchDB interface {
	GetMatch(ctx context.Context, id uuid.UUID) (*Match, error)
	ListMatchesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Match, error)
	ListMatchDetailsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]MatchDetail, error)
	CreateMatch(ctx context.Context, in CreateMatchInput) (*Match, error)
	UpdateMatch(ctx context.Context, in UpdateMatchInput) (*Match, error)
	DeleteMatch(ctx context.Context, id uuid.UUID) error
}

type participantDB interface {
	ListMatchParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error)
	ListParticipantsWithPlayersByTournament(ctx context.Context, tournamentID uuid.UUID) ([]MatchParticipantPlayer, error)
	// SetMatchLineup replaces a match's lineup. guard is the domain's own refusal, run inside
	// the lock and after the repository's, so a scored match says so before anything else.
	SetMatchLineup(ctx context.Context, matchID uuid.UUID, entries []MatchParticipant, guard func() error) error
}

type scoreDB interface {
	ListScoresByMatch(ctx context.Context, matchID uuid.UUID) ([]Score, error)
	ListScoresByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Score, error)
	ResetMatch(ctx context.Context, matchID uuid.UUID) error
	// SaveScoresAndRecompute upserts a hole's scores (per-player when PlayerID is set,
	// else one team row) and rewrites the match's stored result, all in one transaction,
	// returning the recomputed state. Every score lands or none does, and the repo
	// serializes concurrent submissions on a match so neither lands a stale result. guard
	// sees the match's scores before the write and can refuse it; recompute sees them after.
	SaveScoresAndRecompute(
		ctx context.Context,
		matchID uuid.UUID,
		scores []Score,
		tournamentID uuid.UUID,
		guard func(before []Score) error,
		recompute func(after []Score) MatchState,
	) (MatchState, error)
}

// holeDB reads a tee set's holes (course setup).
type holeDB interface {
	ListHolesByTeeSet(ctx context.Context, courseID, teeColorID uuid.UUID) ([]Hole, error)
}

type teamDB interface {
	GetTeam(ctx context.Context, id uuid.UUID) (*Team, error)
	// ListTeamsByTournament returns the tournament's teams with their captains resolved.
	ListTeamsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]TeamWithCaptain, error)
	// SetTeamCaptain assigns a team's captain and returns the updated team.
	SetTeamCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*Team, error)
	// ClearCaptainForPlayer clears the player as a team's captain if they are it (no-op otherwise).
	ClearCaptainForPlayer(ctx context.Context, teamID, playerID uuid.UUID) error
	// ClearCaptain unsets a team's captain outright; ErrTeamNotFound if the team doesn't exist.
	ClearCaptain(ctx context.Context, teamID uuid.UUID) error
}

type teamMemberDB interface {
	// CreateTeamMember drafts a player onto a team (the tournament is the team's).
	CreateTeamMember(ctx context.Context, teamID, playerID, tournamentID uuid.UUID) (*TeamMember, error)
	// DeleteTeamMember undrafts a player; ErrTeamMemberNotFound if they weren't on the team.
	DeleteTeamMember(ctx context.Context, teamID, playerID uuid.UUID) error
}

type tournamentPlayerDB interface {
	CreateTournamentPlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error)
	UpdateTournamentPlayer(ctx context.Context, in UpdateRosterEntryInput) (*TournamentPlayer, error)
	ListTournamentPlayers(ctx context.Context, tournamentID uuid.UUID) ([]TournamentPlayer, error)
	ListTournamentPlayersByTeam(ctx context.Context, teamID uuid.UUID) ([]TournamentPlayer, error)
}

type teeColorDB interface {
	CreateTeeColor(ctx context.Context, in CreateTeeColorInput) (*TeeColor, error)
	ListTeeColors(ctx context.Context) ([]TeeColor, error)
}

type courseDB interface {
	CreateCourse(ctx context.Context, in CreateCourseInput) (*Course, error)
	GetCourse(ctx context.Context, id uuid.UUID) (*Course, error)
	ListCourses(ctx context.Context) ([]Course, error)
}

type teeSetDB interface {
	CreateTeeSet(ctx context.Context, in CreateTeeSetInput) (*TeeSetWithHoles, error)
	// ListTeeSetsByCourse returns a course's tee sets with their colour name resolved.
	ListTeeSetsByCourse(ctx context.Context, courseID uuid.UUID) ([]CourseTeeSet, error)
}

type formatDB interface {
	GetMatchFormat(ctx context.Context, id uuid.UUID) (*MatchFormat, error)
	ListMatchFormats(ctx context.Context) ([]MatchFormat, error)
}

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
	GetMatchResult(ctx context.Context, matchID uuid.UUID) (*StoredResult, error)
	// ListMatchOutcomes returns one entry per match in the tournament; the standings
	// rules that consume them live in standings.go.
	ListMatchOutcomes(ctx context.Context, tournamentID uuid.UUID) ([]MatchOutcome, error)
	// The same, for every tournament at once and keyed by tournament, so listing cups
	// costs one query rather than one per cup.
	ListAllMatchOutcomes(ctx context.Context) (map[uuid.UUID][]MatchOutcome, error)
	// Batched over a tournament's roster, keyed by player id.
	ListTournamentPlayerRecords(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]PlayerRecord, error)
	// The raw material for cups won and a player's history verdict, both decided in
	// standings.go rather than in SQL. Each is a single round trip.
	ListAllTournamentStandings(ctx context.Context) (map[uuid.UUID]TournamentStandings, error)
	ListCupData(ctx context.Context) (CupData, error)
}
