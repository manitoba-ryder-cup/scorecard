package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	"github.com/travisbale/knowhere/identity"
	"github.com/travisbale/knowhere/jwt"
)

// An interface so the health check can verify readiness without this package importing
// the persistence layer. Satisfied by knowhere's generic *postgres.DB.
type database interface {
	Health(ctx context.Context) error
}

// Router carries everything the handlers reach. The domain services are the concrete
// types: the API layer is a translation layer over the domain and has no reason to
// abstract it.
type Router struct {
	DB               database
	JWTValidator     *jwt.Validator
	TrustedProxyMode bool // Trust X-Forwarded-For headers from reverse proxy
	// ProxySecret, when set, requires a matching X-Proxy-Secret header on every request
	// except the health check, so only traffic through the trusted edge (the Cloudflare
	// Worker) is served. Empty disables the check (local dev).
	ProxySecret string
	// PublicTenantID enables anonymous read access for a single-tenant public site
	// (e.g. manitobarydercup.com): reads without a token resolve to this tenant. Nil
	// on a multi-tenant deployment, where every request must carry a token.
	PublicTenantID    *uuid.UUID
	PlayerService     *golf.PlayerService
	MatchService      *golf.MatchService
	TournamentService *golf.TournamentService
	CourseService     *golf.CourseService
	FormatService     *golf.FormatService
	RosterService     *golf.RosterService
	TeamService       *golf.TeamService

	jwtMiddleware *jwt.HTTPMiddleware
}

// Handler assembles the routes and the middleware chain around them. The caller building
// the server calls it once.
func (r *Router) Handler() http.Handler {
	r.jwtMiddleware = jwt.NewHTTPMiddleware(r.JWTValidator)

	mux := http.NewServeMux()
	r.registerRoutes(mux)

	var handler http.Handler = mux
	handler = identity.UserAgent(handler)
	handler = identity.ClientIP(r.TrustedProxyMode)(handler)
	handler = identity.RequestID(handler)
	handler = identity.RequireProxySecret(r.ProxySecret, sdk.RouteHealth)(handler)
	handler = recoverMiddleware(handler)

	return handler
}

// registerRoutes configures every route with its authentication and caching wrapper.
func (r *Router) registerRoutes(mux *http.ServeMux) {

	// Health check (public, no auth, no tenant) — verifies DB readiness.
	mux.HandleFunc("GET "+sdk.RouteHealth, r.handleHealth)

	// Match formats are global seeded reference data — truly public, no tenant needed.
	mux.HandleFunc("GET "+sdk.RouteV1MatchFormats, cacheableRead(r.listMatchFormats))

	// public registers a read that falls back to the public tenant when there is no token.
	public := func(method, route string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, cacheableRead(optionalAuth(r.jwtMiddleware, r.PublicTenantID, handler)))
	}
	// scoped registers a write route that requires a valid token carrying `scope`.
	scoped := func(method, route, scope string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, r.jwtMiddleware.RequireScope(jwt.Scope(scope), handler))
	}

	// Player routes
	public("GET", sdk.RouteV1Players, r.listPlayers)
	scoped("POST", sdk.RouteV1Players, sdk.ScopePlayersWrite, r.createPlayer)
	scoped("PUT", sdk.RouteV1Player, sdk.ScopePlayersWrite, r.updatePlayer)
	public("GET", sdk.RouteV1Player, r.getPlayer)
	public("GET", sdk.RouteV1PlayerTournaments, r.listPlayerTournaments)
	public("GET", sdk.RouteV1PlayerStats, r.getPlayerStats)

	// Course reference-data routes
	public("GET", sdk.RouteV1TeeColors, r.listTeeColors)
	scoped("POST", sdk.RouteV1TeeColors, sdk.ScopeCoursesWrite, r.createTeeColor)
	public("GET", sdk.RouteV1Courses, r.listCourses)
	scoped("POST", sdk.RouteV1Courses, sdk.ScopeCoursesWrite, r.createCourse)
	public("GET", sdk.RouteV1Course, r.getCourse)
	public("GET", sdk.RouteV1CourseTees, r.listCourseTeeSets)
	scoped("POST", sdk.RouteV1CourseTees, sdk.ScopeCoursesWrite, r.addTeeSet)

	// Match routes
	public("GET", sdk.RouteV1MatchParticipants, r.listParticipants)
	scoped("POST", sdk.RouteV1MatchParticipants, sdk.ScopeTournamentsWrite, r.addParticipant)
	scoped("DELETE", sdk.RouteV1MatchParticipant, sdk.ScopeTournamentsWrite, r.removeParticipant)
	public("GET", sdk.RouteV1MatchScores, r.getMatchScores)
	public("GET", sdk.RouteV1MatchHoles, r.getMatchHoles)
	scoped("POST", sdk.RouteV1MatchScores, sdk.ScopeScoresWrite, r.submitScore)
	// Scoring is the one grant handed to someone on the course; clearing a match is not.
	scoped("DELETE", sdk.RouteV1MatchScores, sdk.ScopeTournamentsWrite, r.resetMatch)
	public("GET", sdk.RouteV1MatchWinner, r.getMatchStatus)
	public("GET", sdk.RouteV1MatchStatus, r.getMatchStatus)

	// Tournament routes
	public("GET", sdk.RouteV1Tournaments, r.listTournaments)
	scoped("POST", sdk.RouteV1Tournaments, sdk.ScopeTournamentsWrite, r.createTournament)
	public("GET", sdk.RouteV1Tournament, r.getTournament)
	public("GET", sdk.RouteV1TournamentTeams, r.getTournamentTeams)
	public("GET", sdk.RouteV1TournamentResults, r.listResults)

	// Match setup routes (matches live under a tournament)
	public("GET", sdk.RouteV1TournamentMatches, r.listMatches)
	scoped("POST", sdk.RouteV1TournamentMatches, sdk.ScopeTournamentsWrite, r.createMatch)
	scoped("PUT", sdk.RouteV1Match, sdk.ScopeTournamentsWrite, r.updateMatch)
	scoped("DELETE", sdk.RouteV1Match, sdk.ScopeTournamentsWrite, r.deleteMatch)

	// Tournament roster routes
	public("GET", sdk.RouteV1TournamentPlayers, r.listTournamentPlayers)
	scoped("POST", sdk.RouteV1TournamentPlayers, sdk.ScopeTournamentsWrite, r.enterPlayer)
	scoped("PUT", sdk.RouteV1TournamentPlayer, sdk.ScopeTournamentsWrite, r.updateTournamentPlayer)
	public("GET", sdk.RouteV1TournamentWinner, r.getTournamentWinner)
	public("GET", sdk.RouteV1TournamentStatus, r.getTournamentStatus)

	// Team draft routes
	public("GET", sdk.RouteV1TeamMembers, r.listTeamMembers)
	scoped("POST", sdk.RouteV1TeamMembers, sdk.ScopeTournamentsWrite, r.draftPlayer)
	scoped("DELETE", sdk.RouteV1TeamMember, sdk.ScopeTournamentsWrite, r.undraftPlayer)
	scoped("PUT", sdk.RouteV1TeamCaptain, sdk.ScopeTournamentsWrite, r.setCaptain)
	scoped("DELETE", sdk.RouteV1TeamCaptain, sdk.ScopeTournamentsWrite, r.clearCaptain)

	// Assembled inner-to-outer, so the last one wrapped is the first to run.
}
