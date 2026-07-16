package main

import (
	"log/slog"

	"github.com/manitoba-ryder-cup/scorecard/internal/app"
)

// Config holds all configuration for the application
type Config struct {
	// Debug
	Debug bool

	// Database
	DatabaseURL string

	// Server address
	HTTPAddress string

	// JWT configuration
	JWTPublicKeyPath string

	// Environment
	Environment string

	// Proxy configuration
	TrustedProxyMode bool

	// PublicTenantID enables anonymous public reads scoped to this tenant (empty on a
	// multi-tenant deployment)
	PublicTenantID string
}

// config is the global configuration populated by CLI flags
var config = &Config{}

// ToAppConfig converts the CLI config to an app.Config
func (c *Config) ToAppConfig() *app.Config {
	return &app.Config{
		DatabaseURL:      c.DatabaseURL,
		HTTPAddress:      c.HTTPAddress,
		JWTPublicKeyPath: c.JWTPublicKeyPath,
		Environment:      c.Environment,
		TrustedProxyMode: c.TrustedProxyMode,
		PublicTenantID:   c.PublicTenantID,
		Logger:           slog.Default(),
	}
}
