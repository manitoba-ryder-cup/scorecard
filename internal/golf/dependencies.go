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

// tournamentDB interface defines database operations for tournaments
type tournamentDB interface {
	GetTournament(ctx context.Context, id int32) (*Tournament, error)
	ListTournaments(ctx context.Context) ([]Tournament, error)
}
