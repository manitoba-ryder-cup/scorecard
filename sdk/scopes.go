package sdk

// API write scopes. Each mutating endpoint requires the matching scope in the
// caller's token (issued by heimdall). Reads are public — either scoped to the
// token's tenant when one is present, or to the deployment's configured public
// tenant for anonymous spectators. These strings are the contract heimdall grants
// against and scorecard enforces.
const (
	ScopeTournamentsWrite = "scorecard:tournaments:write"
	ScopePlayersWrite     = "scorecard:players:write"
	ScopeScoresWrite      = "scorecard:scores:write"
)
