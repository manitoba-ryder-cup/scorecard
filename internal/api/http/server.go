package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/travisbale/knowhere/identity"
	"github.com/travisbale/knowhere/jwt"
	"github.com/travisbale/scorecard/internal/db/postgres"
	"github.com/travisbale/scorecard/internal/golf"
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
	router := chi.NewRouter()

	// Global middleware
	router.Use(middleware.Recoverer)
	router.Use(identity.RequestID)
	router.Use(identity.ClientIP(config.TrustedProxyMode))
	router.Use(identity.UserAgent)

	// Health check endpoint (public, no auth required)
	router.Get("/healthz", HandleHealth)

	// Create JWT middleware
	jwtMiddleware := jwt.NewHTTPMiddleware(config.JWTValidator)

	// Create handlers
	playersHandler := NewPlayersHandler(config.PlayerService)
	matchesHandler := NewMatchesHandler(config.MatchService)
	tournamentsHandler := NewTournamentsHandler(config.TournamentService)

	// API v1 routes (protected with JWT authentication)
	router.Group(func(protected chi.Router) {
		protected.Use(jwtMiddleware.Authenticate)

		// Player routes
		protected.Get("/v1/players", playersHandler.ListPlayers)
		protected.Get("/v1/players/{id}", playersHandler.GetPlayer)

		// Match routes
		protected.Get("/v1/matches/{id}/scores", matchesHandler.GetMatchScores)
		protected.Get("/v1/matches/{id}/winner", matchesHandler.GetMatchWinner)
		protected.Get("/v1/matches/{id}/status", matchesHandler.GetMatchStatus)

		// Tournament routes
		protected.Get("/v1/tournaments", tournamentsHandler.ListTournaments)
		protected.Get("/v1/tournaments/{id}", tournamentsHandler.GetTournament)
		protected.Get("/v1/tournaments/{id}/teams", tournamentsHandler.GetTournamentTeams)
		protected.Get("/v1/tournaments/{id}/winner", tournamentsHandler.GetTournamentWinner)
		protected.Get("/v1/tournaments/{id}/status", tournamentsHandler.GetTournamentStatus)
	})

	return &Server{
		&http.Server{
			Addr:              config.Address,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
