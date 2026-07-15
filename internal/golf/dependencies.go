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
	// Returns participants with their team assignment
	ListMatchParticipantsWithTeam(ctx context.Context, matchID int32) (map[int32]string, error) // playerID -> teamName
}

// scoreDB interface defines database operations for scores
type scoreDB interface {
	ListScoresByMatch(ctx context.Context, matchID int32) ([]Score, error)
	ListScoresByMatchAndPlayer(ctx context.Context, matchID int32, playerID int32) ([]Score, error)
	// Returns hole handicap for a score
	GetHoleHandicap(ctx context.Context, courseID int32, teeColorID int32, holeNumber int32) (int32, error)
}

// teamDB interface defines database operations for teams
type teamDB interface {
	GetTeam(ctx context.Context, id int32) (*Team, error)
	ListTeams(ctx context.Context) ([]Team, error)
}

// teamMemberDB interface defines database operations for team members
type teamMemberDB interface {
	ListTeamMembers(ctx context.Context, tournamentID int32, teamID int32) ([]TeamMember, error)
	// Returns the captain for a team in a tournament
	GetTeamCaptain(ctx context.Context, tournamentID int32, teamID int32) (*Player, error)
}

// tournamentDB interface defines database operations for tournaments
type tournamentDB interface {
	GetTournament(ctx context.Context, id int32) (*Tournament, error)
	ListTournaments(ctx context.Context) ([]Tournament, error)
}
