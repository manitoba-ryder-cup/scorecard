package sdk

// Team colors. A Ryder Cup tournament has exactly two sides, one of each color,
// seeded automatically when the tournament is created (the database also enforces
// this with a CHECK and a UNIQUE(tournament_id, color)). These constants are the
// single source of truth shared by the domain and the wire layer.
const (
	TeamColorRed  = "Red"
	TeamColorBlue = "Blue"
)

// DefaultTier is applied when a tournament entry omits one, mirroring the schema
// default on tournament_players.tier.
const DefaultTier = "white"

// Player-tournament outcomes
type Result string

const (
	ResultWon        Result = "won"
	ResultLost       Result = "lost"
	ResultTied       Result = "tied"
	ResultInProgress Result = "in_progress"
)

// TournamentPhase is where a cup stands in its life. A cup sits in PhaseUpcoming for the
// months between its roster being entered and its first tee time, so "not finished" alone
// does not mean anything is moving — which is why this is published rather than left for
// each client to re-derive from the results.
type TournamentPhase string

const (
	PhaseUpcoming TournamentPhase = "upcoming"
	PhaseLive     TournamentPhase = "live"
	PhaseFinished TournamentPhase = "finished"
)
