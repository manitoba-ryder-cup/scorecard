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

// HealthChecker reports whether a dependency (the database) is reachable. Satisfied by
// knowhere's generic *postgres.DB, so /health can verify readiness without this
// package importing the persistence layer.
type HealthChecker interface {
	Health(ctx context.Context) error
}

// Router carries everything the handlers reach. The domain services are the concrete
// types: the API layer is a translation layer over the domain and has no reason to
// abstract it.
type Router struct {
	// DB answers the health check. An interface so /health can verify readiness without
	// this package importing the persistence layer.
	DB               HealthChecker
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
	mux.HandleFunc("GET "+sdk.RouteHealth, HandleHealth(r.DB))

	// Match formats are global seeded reference data — truly public, no tenant needed.
	mux.HandleFunc("GET "+sdk.RouteV1MatchFormats, cacheableRead(r.ListMatchFormats))

	// public registers a read route with optional authentication: a token's tenant is
	// used when present, else the configured public tenant (401 if neither). Anonymous
	// successes are marked edge-cacheable on the way out — see cacheableRead.
	public := func(method, route string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, cacheableRead(optionalAuth(r.jwtMiddleware, r.PublicTenantID, handler)))
	}
	// scoped registers a write route that requires a valid token carrying `scope`.
	scoped := func(method, route, scope string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, r.jwtMiddleware.RequireScope(jwt.Scope(scope), handler))
	}

	// Player routes
	public("GET", sdk.RouteV1Players, r.ListPlayers)
	scoped("POST", sdk.RouteV1Players, sdk.ScopePlayersWrite, r.CreatePlayer)
	scoped("PUT", sdk.RouteV1Player, sdk.ScopePlayersWrite, r.UpdatePlayer)
	public("GET", sdk.RouteV1Player, r.GetPlayer)
	public("GET", sdk.RouteV1PlayerTournaments, r.ListPlayerTournaments)
	public("GET", sdk.RouteV1PlayerStats, r.GetPlayerStats)

	// Course reference-data routes
	public("GET", sdk.RouteV1TeeColors, r.ListTeeColors)
	scoped("POST", sdk.RouteV1TeeColors, sdk.ScopeCoursesWrite, r.CreateTeeColor)
	public("GET", sdk.RouteV1Courses, r.ListCourses)
	scoped("POST", sdk.RouteV1Courses, sdk.ScopeCoursesWrite, r.CreateCourse)
	public("GET", sdk.RouteV1Course, r.GetCourse)
	public("GET", sdk.RouteV1CourseTees, r.ListCourseTeeSets)
	scoped("POST", sdk.RouteV1CourseTees, sdk.ScopeCoursesWrite, r.AddTeeSet)

	// Match routes
	public("GET", sdk.RouteV1MatchParticipants, r.ListParticipants)
	scoped("POST", sdk.RouteV1MatchParticipants, sdk.ScopeTournamentsWrite, r.AddParticipant)
	scoped("DELETE", sdk.RouteV1MatchParticipant, sdk.ScopeTournamentsWrite, r.RemoveParticipant)
	public("GET", sdk.RouteV1MatchScores, r.GetMatchScores)
	public("GET", sdk.RouteV1MatchHoles, r.GetMatchHoles)
	scoped("POST", sdk.RouteV1MatchScores, sdk.ScopeScoresWrite, r.SubmitScore)
	public("GET", sdk.RouteV1MatchWinner, r.GetMatchStatus)
	public("GET", sdk.RouteV1MatchStatus, r.GetMatchStatus)

	// Tournament routes
	public("GET", sdk.RouteV1Tournaments, r.ListTournaments)
	scoped("POST", sdk.RouteV1Tournaments, sdk.ScopeTournamentsWrite, r.CreateTournament)
	public("GET", sdk.RouteV1Tournament, r.GetTournament)
	public("GET", sdk.RouteV1TournamentTeams, r.GetTournamentTeams)
	public("GET", sdk.RouteV1TournamentResults, r.ListResults)

	// Match setup routes (matches live under a tournament)
	public("GET", sdk.RouteV1TournamentMatches, r.ListMatches)
	scoped("POST", sdk.RouteV1TournamentMatches, sdk.ScopeTournamentsWrite, r.CreateMatch)
	scoped("PUT", sdk.RouteV1Match, sdk.ScopeTournamentsWrite, r.UpdateMatch)

	// Tournament roster routes
	public("GET", sdk.RouteV1TournamentPlayers, r.ListTournamentPlayers)
	scoped("POST", sdk.RouteV1TournamentPlayers, sdk.ScopeTournamentsWrite, r.EnterPlayer)
	scoped("PUT", sdk.RouteV1TournamentPlayer, sdk.ScopeTournamentsWrite, r.UpdateTournamentPlayer)
	public("GET", sdk.RouteV1TournamentWinner, r.GetTournamentWinner)
	public("GET", sdk.RouteV1TournamentStatus, r.GetTournamentStatus)

	// Team draft routes
	public("GET", sdk.RouteV1TeamMembers, r.ListTeamMembers)
	scoped("POST", sdk.RouteV1TeamMembers, sdk.ScopeTournamentsWrite, r.DraftPlayer)
	scoped("DELETE", sdk.RouteV1TeamMember, sdk.ScopeTournamentsWrite, r.UndraftPlayer)
	scoped("PUT", sdk.RouteV1TeamCaptain, sdk.ScopeTournamentsWrite, r.SetCaptain)
	scoped("DELETE", sdk.RouteV1TeamCaptain, sdk.ScopeTournamentsWrite, r.ClearCaptain)

	// Global middleware chain. Assembled inner-to-outer, so recoverMiddleware is
	// outermost (wraps everything) and RequestID runs before ClientIP/UserAgent.
}
