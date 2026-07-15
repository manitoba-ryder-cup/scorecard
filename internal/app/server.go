package app

import (
	"context"
	"fmt"

	"github.com/travisbale/knowhere/jwt"
	"github.com/manitoba-ryder-cup/scorecard/internal/api/http"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Config holds the configuration for creating a new server
type Config struct {
	HTTPAddress      string
	DatabaseURL      string
	JWTPublicKeyPath string
	Environment      string
	TrustedProxyMode bool
	Logger           logger
}

// Server wraps the HTTP server and its dependencies
type Server struct {
	httpServer *http.Server
	db         *postgres.DB
}

// NewServer creates a new server instance with all dependencies
func NewServer(ctx context.Context, config *Config) (*Server, error) {
	// Connect to database using knowhere wrapper
	db, err := postgres.NewDB(ctx, config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run database migrations
	if err := postgres.MigrateUp(config.DatabaseURL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Create JWT validator
	jwtValidator, err := jwt.NewValidator(config.JWTPublicKeyPath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create JWT validator: %w", err)
	}

	// Create database adapters
	playersDB := postgres.NewPlayersDB(db)
	matchesDB := postgres.NewMatchesDB(db)
	participantsDB := postgres.NewParticipantsDB(db)
	scoresDB := postgres.NewScoresDB(db)
	teamsDB := postgres.NewTeamsDB(db)
	teamMembersDB := postgres.NewTeamMembersDB(db)
	tournamentsDB := postgres.NewTournamentsDB(db)

	// Create domain services
	playerService := &golf.PlayerService{
		PlayerDB: playersDB,
		Logger:   config.Logger,
	}

	matchService := &golf.MatchService{
		MatchDB:       matchesDB,
		ParticipantDB: participantsDB,
		ScoreDB:       scoresDB,
		PlayerDB:      playersDB,
		Logger:        config.Logger,
	}

	teamService := &golf.TeamService{
		TeamDB:       teamsDB,
		TeamMemberDB: teamMembersDB,
		MatchService: matchService,
		Logger:       config.Logger,
	}

	tournamentService := &golf.TournamentService{
		TournamentDB: tournamentsDB,
		MatchService: matchService,
		TeamService:  teamService,
		Logger:       config.Logger,
	}

	// Create HTTP server
	httpServer := http.NewServer(&http.Config{
		Address:           config.HTTPAddress,
		JWTValidator:      jwtValidator,
		DB:                db,
		Environment:       config.Environment,
		TrustedProxyMode:  config.TrustedProxyMode,
		PlayerService:     playerService,
		MatchService:      matchService,
		TeamService:       teamService,
		TournamentService: tournamentService,
	})

	return &Server{
		httpServer: httpServer,
		db:         db,
	}, nil
}

// Start begins listening for HTTP requests
func (s *Server) Start() error {
	// Start HTTP server (blocking)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Close database connection
	s.db.Close()

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}
