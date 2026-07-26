package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/api/rest"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
	"github.com/travisbale/knowhere/jwt"
)

// Config holds the configuration for creating a new server
type Config struct {
	HTTPAddress      string
	DatabaseURL      string
	JWTPublicKeyPath string
	Environment      string
	TrustedProxyMode bool
	// ProxySecret, when set, requires an X-Proxy-Secret header on every request except
	// the health check (only the trusted edge is served). Empty disables the check.
	ProxySecret string
	// PublicTenantID, when set, enables anonymous read access scoped to that tenant
	// (a single-tenant public site). Empty on a multi-tenant deployment.
	PublicTenantID string
}

// drainer is satisfied by *rest.Server.
type drainer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// closer is satisfied by *postgres.DB.
type closer interface {
	Close()
}

// Server wraps the HTTP server and its dependencies. Held as interfaces so the
// shutdown ordering is testable without real infrastructure.
type Server struct {
	httpServer drainer
	db         closer
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

	// Wire the domain services (shared with the CLI commands).
	services := NewServices(db)

	// Create HTTP server
	httpServer := rest.NewServer(&rest.Config{
		Address:           config.HTTPAddress,
		JWTValidator:      jwtValidator,
		TrustedProxyMode:  config.TrustedProxyMode,
		ProxySecret:       config.ProxySecret,
		PublicTenantID:    publicTenantID,
		DB:                db,
		PlayerService:     services.Player,
		MatchService:      services.Match,
		TournamentService: services.Tournament,
		CourseService:     services.Course,
		FormatService:     services.Format,
		RosterService:     services.Roster,
		TeamService:       services.Team,
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

// Shutdown drains in-flight requests before releasing the pool. The order matters:
// Shutdown blocks until active handlers return, and those handlers are still querying.
// The deferred Close runs even if draining fails, so a timeout can't leak connections.
func (s *Server) Shutdown(ctx context.Context) error {
	defer s.db.Close()
	return s.httpServer.Shutdown(ctx)
}
