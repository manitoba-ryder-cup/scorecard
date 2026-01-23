package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/travisbale/knowhere/jwt"
	"github.com/travisbale/scorecard/internal/db/postgres"
)

type Config struct {
	Address      string
	JWTValidator *jwt.Validator
	DB           *postgres.DB
	Environment  string // "development", "staging", "production"
}

type Server struct {
	*http.Server
}

func NewServer(config *Config) *Server {
	router := chi.NewRouter()

	// Global middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)

	// Health check endpoint (public, no auth required)
	router.Get("/healthz", HandleHealth)

	// API v1 routes
	router.Route("/v1", func(router chi.Router) {
		// Add your authenticated routes here
		// Example:
		// router.Group(func(r chi.Router) {
		//     r.Use(AuthMiddleware(config.JWTValidator))
		//     r.Get("/resource", HandleGetResource)
		// })
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
