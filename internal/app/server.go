package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/api/rest"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/travisbale/knowhere/jwt"
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
	// PublicTenantID, when set, enables anonymous read access scoped to that tenant
	// (a single-tenant public site). Empty on a multi-tenant deployment.
	PublicTenantID string
	Logger         logger
}

// Server wraps the HTTP server and its dependencies
type Server struct {
	httpServer *rest.Server
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

	// Parse the optional public tenant (fail fast on a malformed value rather than
	// silently disabling public reads).
	var publicTenantID *uuid.UUID
	if config.PublicTenantID != "" {
		id, err := uuid.Parse(config.PublicTenantID)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("invalid public tenant ID %q: %w", config.PublicTenantID, err)
		}
		publicTenantID = &id
	}

	// Create database adapters
	playersDB := postgres.NewPlayersDB(db)
	matchesDB := postgres.NewMatchesDB(db)
	participantsDB := postgres.NewParticipantsDB(db)
	scoresDB := postgres.NewScoresDB(db)
	teamsDB := postgres.NewTeamsDB(db)
	teamMembersDB := postgres.NewTeamMembersDB(db)
	tournamentsDB := postgres.NewTournamentsDB(db)
	resultsDB := postgres.NewResultsDB(db)
	teeColorsDB := postgres.NewTeeColorsDB(db)
	coursesDB := postgres.NewCoursesDB(db)
	teeSetsDB := postgres.NewTeeSetsDB(db)

	// Create domain services
	playerService := &golf.PlayerService{
		PlayerDB: playersDB,
		ResultDB: resultsDB,
		Logger:   config.Logger,
	}

	matchService := &golf.MatchService{
		MatchDB:       matchesDB,
		ParticipantDB: participantsDB,
		ScoreDB:       scoresDB,
		ResultDB:      resultsDB,
		Logger:        config.Logger,
	}

	teamService := &golf.TeamService{
		TeamDB:       teamsDB,
		TeamMemberDB: teamMembersDB,
		Logger:       config.Logger,
	}

	tournamentService := &golf.TournamentService{
		TournamentDB: tournamentsDB,
		ResultDB:     resultsDB,
		TeamService:  teamService,
		Logger:       config.Logger,
	}

	courseService := &golf.CourseService{
		TeeColorDB: teeColorsDB,
		CourseDB:   coursesDB,
		TeeSetDB:   teeSetsDB,
		Logger:     config.Logger,
	}

	// Create HTTP server
	httpServer := rest.NewServer(&rest.Config{
		Address:           config.HTTPAddress,
		JWTValidator:      jwtValidator,
		Environment:       config.Environment,
		TrustedProxyMode:  config.TrustedProxyMode,
		PublicTenantID:    publicTenantID,
		PlayerService:     playerService,
		MatchService:      matchService,
		TournamentService: tournamentService,
		CourseService:     courseService,
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
