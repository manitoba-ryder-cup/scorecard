package sdk

// API route constants shared between the server and SDK clients.
const (
	RouteHealth = "/healthz"

	// Players
	RouteV1Players = "/v1/players"
	RouteV1Player  = "/v1/players/{id}"

	// Reference data
	RouteV1MatchFormats = "/v1/match-formats"

	// Course reference data
	RouteV1TeeColors  = "/v1/tee-colors"
	RouteV1Courses    = "/v1/courses"
	RouteV1Course     = "/v1/courses/{id}"
	RouteV1CourseTees = "/v1/courses/{id}/tees"

	// Tournaments
	RouteV1Tournaments      = "/v1/tournaments"
	RouteV1Tournament       = "/v1/tournaments/{id}"
	RouteV1TournamentWinner = "/v1/tournaments/{id}/winner"
	RouteV1TournamentStatus = "/v1/tournaments/{id}/status"

	// Roster (players entered in a tournament)
	RouteV1TournamentPlayers = "/v1/tournaments/{id}/players"
	RouteV1TournamentPlayer  = "/v1/tournaments/{id}/players/{playerId}"

	// Teams (scoped to a tournament)
	RouteV1TournamentTeams = "/v1/tournaments/{id}/teams"

	// Matches
	RouteV1TournamentMatches = "/v1/tournaments/{id}/matches"
	RouteV1Match             = "/v1/matches/{id}"
	RouteV1MatchScores       = "/v1/matches/{id}/scores"
	RouteV1MatchWinner       = "/v1/matches/{id}/winner"
	RouteV1MatchStatus       = "/v1/matches/{id}/status"
)
