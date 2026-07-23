package sdk

// Team colors. A Ryder Cup tournament has exactly two sides, one of each color,
// seeded automatically when the tournament is created (the database also enforces
// this with a CHECK and a UNIQUE(tournament_id, color)). These constants are the
// single source of truth shared by the domain and the wire layer.
const (
	TeamColorRed  = "Red"
	TeamColorBlue = "Blue"
)

// Player-tournament outcomes (PlayerTournamentHistory.result). "in_progress" covers a
// tournament that hasn't finished all its matches yet.
const (
	ResultWon        = "won"
	ResultLost       = "lost"
	ResultTied       = "tied"
	ResultInProgress = "in_progress"
)
