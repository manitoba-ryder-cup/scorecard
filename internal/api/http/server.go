package http

import (
	"context"
	"net/http"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/identity"
	"github.com/travisbale/knowhere/jwt"
)

type Config struct {
	Address           string
	JWTValidator      *jwt.Validator
	DB                *postgres.DB
	Environment       string // "development", "staging", "production"
	TrustedProxyMode  bool   // Trust X-Forwarded-For headers from reverse proxy
	PlayerService     *golf.PlayerService
	MatchService      *golf.MatchService
	TeamService       *golf.TeamService
	TournamentService *golf.TournamentService
}

type Server struct {
	*http.Server
}

func NewServer(config *Config) *Server {
	jwtMiddleware := jwt.NewHTTPMiddleware(config.JWTValidator)

	playersHandler := NewPlayersHandler(config.PlayerService)
	matchesHandler := NewMatchesHandler(config.MatchService)
	tournamentsHandler := NewTournamentsHandler(config.TournamentService)

	mux := http.NewServeMux()

	// Health check (public, no auth)
	mux.HandleFunc("GET /healthz", HandleHealth)

	// auth wraps a handler with JWT authentication. knowhere's Authenticate is
	// func(http.HandlerFunc) http.HandlerFunc — the std-lib shape used directly
	// with ServeMux (chi's Handler-based Use rejected this signature).
	auth := func(method, route string, handler http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, jwtMiddleware.Authenticate(handler))
	}

	// Player routes
	auth("GET", "/v1/players", playersHandler.ListPlayers)
	auth("GET", "/v1/players/{id}", playersHandler.GetPlayer)

	// Match routes
	auth("GET", "/v1/matches/{id}/scores", matchesHandler.GetMatchScores)
	auth("GET", "/v1/matches/{id}/winner", matchesHandler.GetMatchWinner)
	auth("GET", "/v1/matches/{id}/status", matchesHandler.GetMatchStatus)

	// Tournament routes
	auth("GET", "/v1/tournaments", tournamentsHandler.ListTournaments)
	auth("GET", "/v1/tournaments/{id}", tournamentsHandler.GetTournament)
	auth("GET", "/v1/tournaments/{id}/teams", tournamentsHandler.GetTournamentTeams)
	auth("GET", "/v1/tournaments/{id}/winner", tournamentsHandler.GetTournamentWinner)
	auth("GET", "/v1/tournaments/{id}/status", tournamentsHandler.GetTournamentStatus)

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

// recoverMiddleware turns a panic in a downstream handler into a 500 instead of
// crashing the server. knowhere provides no recoverer; this mirrors heimdall.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
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
