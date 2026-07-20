package rest

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	"github.com/travisbale/knowhere/identity"
	"github.com/travisbale/knowhere/jwt"
)

// HealthChecker reports whether a dependency (the database) is reachable. Satisfied by
// knowhere's generic *postgres.DB, so /healthz can verify readiness without this
// package importing the persistence layer.
type HealthChecker interface {
	Health(ctx context.Context) error
}

type Config struct {
	Address          string
	JWTValidator     *jwt.Validator
	TrustedProxyMode bool // Trust X-Forwarded-For headers from reverse proxy
	// PublicTenantID enables anonymous read access for a single-tenant public site
	// (e.g. manitobarydercup.com): reads without a token resolve to this tenant. Nil
	// on a multi-tenant deployment, where every request must carry a token.
	PublicTenantID    *uuid.UUID
	DB                HealthChecker
	PlayerService     *golf.PlayerService
	MatchService      *golf.MatchService
	TournamentService *golf.TournamentService
	CourseService     *golf.CourseService
	FormatService     *golf.FormatService
	RosterService     *golf.RosterService
	TeamService       *golf.TeamService
}

type Server struct {
	*http.Server
}

func NewServer(config *Config) *Server {
	jwtMiddleware := jwt.NewHTTPMiddleware(config.JWTValidator)

	playersHandler := NewPlayersHandler(config.PlayerService)
	matchesHandler := NewMatchesHandler(config.MatchService)
	tournamentsHandler := NewTournamentsHandler(config.TournamentService)
	coursesHandler := NewCoursesHandler(config.CourseService)
	formatsHandler := NewFormatsHandler(config.FormatService)
	rosterHandler := NewRosterHandler(config.RosterService)
	teamsHandler := NewTeamsHandler(config.TeamService)

	mux := http.NewServeMux()

	// Health check (public, no auth, no tenant) — verifies DB readiness.
	mux.HandleFunc("GET "+sdk.RouteHealth, HandleHealth(config.DB))

	// Match formats are global seeded reference data — truly public, no tenant needed.
	mux.HandleFunc("GET "+sdk.RouteV1MatchFormats, formatsHandler.ListMatchFormats)

	// public registers a read route with optional authentication: a token's tenant is
	// used when present, else the configured public tenant (401 if neither).
	public := func(method, route string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, optionalAuth(jwtMiddleware, config.PublicTenantID, handler))
	}
	// scoped registers a write route that requires a valid token carrying `scope`.
	scoped := func(method, route, scope string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, jwtMiddleware.RequireScope(jwt.Scope(scope), handler))
	}

	// Player routes
	public("GET", "/v1/players", playersHandler.ListPlayers)
	scoped("POST", "/v1/players", sdk.ScopePlayersWrite, playersHandler.CreatePlayer)
	public("GET", "/v1/players/{id}", playersHandler.GetPlayer)

	// Course reference-data routes
	public("GET", "/v1/tee-colors", coursesHandler.ListTeeColors)
	scoped("POST", "/v1/tee-colors", sdk.ScopeCoursesWrite, coursesHandler.CreateTeeColor)
	public("GET", "/v1/courses", coursesHandler.ListCourses)
	scoped("POST", "/v1/courses", sdk.ScopeCoursesWrite, coursesHandler.CreateCourse)
	public("GET", "/v1/courses/{id}", coursesHandler.GetCourse)
	scoped("POST", "/v1/courses/{id}/tees", sdk.ScopeCoursesWrite, coursesHandler.AddTeeSet)

	// Match routes
	public("GET", "/v1/matches/{id}/participants", matchesHandler.ListParticipants)
	scoped("POST", "/v1/matches/{id}/participants", sdk.ScopeTournamentsWrite, matchesHandler.AddParticipant)
	public("GET", "/v1/matches/{id}/scores", matchesHandler.GetMatchScores)
	scoped("POST", "/v1/matches/{id}/scores", sdk.ScopeScoresWrite, matchesHandler.SubmitScore)
	public("GET", "/v1/matches/{id}/winner", matchesHandler.GetMatchWinner)
	public("GET", "/v1/matches/{id}/status", matchesHandler.GetMatchStatus)

	// Tournament routes
	public("GET", "/v1/tournaments", tournamentsHandler.ListTournaments)
	scoped("POST", "/v1/tournaments", sdk.ScopeTournamentsWrite, tournamentsHandler.CreateTournament)
	public("GET", "/v1/tournaments/{id}", tournamentsHandler.GetTournament)
	public("GET", "/v1/tournaments/{id}/teams", tournamentsHandler.GetTournamentTeams)

	// Match setup routes (matches live under a tournament)
	public("GET", "/v1/tournaments/{id}/matches", matchesHandler.ListMatches)
	scoped("POST", "/v1/tournaments/{id}/matches", sdk.ScopeTournamentsWrite, matchesHandler.CreateMatch)

	// Tournament roster routes
	public("GET", "/v1/tournaments/{id}/players", rosterHandler.ListPlayers)
	scoped("POST", "/v1/tournaments/{id}/players", sdk.ScopeTournamentsWrite, rosterHandler.EnterPlayer)
	scoped("PUT", "/v1/tournaments/{id}/players/{playerId}", sdk.ScopeTournamentsWrite, rosterHandler.UpdatePlayer)
	public("GET", "/v1/tournaments/{id}/winner", tournamentsHandler.GetTournamentWinner)
	public("GET", "/v1/tournaments/{id}/status", tournamentsHandler.GetTournamentStatus)

	// Team draft routes
	public("GET", "/v1/teams/{id}/members", rosterHandler.ListTeamMembers)
	scoped("POST", "/v1/teams/{id}/members", sdk.ScopeTournamentsWrite, rosterHandler.DraftPlayer)
	scoped("PUT", "/v1/teams/{id}/captain", sdk.ScopeTournamentsWrite, teamsHandler.SetCaptain)

	// Global middleware chain. Assembled inner-to-outer, so recoverMiddleware is
	// outermost (wraps everything) and RequestID runs before ClientIP/UserAgent.
	var handler http.Handler = mux
	handler = identity.UserAgent(handler)
	handler = identity.ClientIP(config.TrustedProxyMode)(handler)
	handler = identity.RequestID(handler)
	handler = recoverMiddleware(handler)

	return &Server{
		&http.Server{
			Addr:              config.Address,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// optionalAuth guards a public read route. With an Authorization header it delegates
// to full JWT authentication (tenant + actor from the token; 401 on a bad token).
// Without one, it falls back to the configured public tenant so anonymous spectators
// can read a single-tenant site; if no public tenant is configured, it is 401 (a
// multi-tenant deployment requires login even to read).
func optionalAuth(m *jwt.HTTPMiddleware, publicTenantID *uuid.UUID, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			m.Authenticate(next)(w, r)
			return
		}
		if publicTenantID == nil {
			respondError(r.Context(), w, http.StatusUnauthorized, "authentication required", nil)
			return
		}
		ctx := identity.WithTenant(r.Context(), *publicTenantID)
		next(w, r.WithContext(ctx))
	}
}

// recoverMiddleware turns a panic in a downstream handler into a 500 instead of
// crashing the server. knowhere provides no recoverer; this mirrors heimdall.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log with a stack so a recovered panic leaves a diagnosable trail.
				slog.ErrorContext(r.Context(), "panic recovered", "error", err, "stack", string(debug.Stack()))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
