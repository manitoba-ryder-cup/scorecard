package sdk

// API route constants shared between the server and SDK clients.
const (
	RouteHealth = "/healthz"

	// Players
	RouteV1Players = "/v1/players"
	RouteV1Player  = "/v1/players/{id}"

	// Tournaments
	RouteV1Tournaments = "/v1/tournaments"
	RouteV1Tournament  = "/v1/tournaments/{id}"

	// Teams (scoped to a tournament)
	RouteV1TournamentTeams = "/v1/tournaments/{id}/teams"

	// Matches
	RouteV1TournamentMatches = "/v1/tournaments/{id}/matches"
	RouteV1Match             = "/v1/matches/{id}"
	RouteV1MatchScores       = "/v1/matches/{id}/scores"
)
