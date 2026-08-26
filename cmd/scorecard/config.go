package main

import (
	"github.com/manitoba-ryder-cup/scorecard/internal/app"
)

type Config struct {
	Debug bool

	// LogFormat selects the slog handler: "json" (default) or "text"
	LogFormat string

	DatabaseURL string

	HTTPAddress string

	JWTPublicKeyPath string

	Environment string

	TrustedProxyMode bool

	// ProxySecret gates all non-health requests on a matching X-Proxy-Secret header
	// (set by the trusted edge). Empty disables the check.
	ProxySecret string

	// PublicTenantID enables anonymous public reads scoped to this tenant (empty on a
	// multi-tenant deployment)
	PublicTenantID string
}

// config is the global configuration populated by CLI flags
var config = &Config{}

func (c *Config) ToAppConfig() *app.Config {
	return &app.Config{
		DatabaseURL:      c.DatabaseURL,
		HTTPAddress:      c.HTTPAddress,
		JWTPublicKeyPath: c.JWTPublicKeyPath,
		Environment:      c.Environment,
		TrustedProxyMode: c.TrustedProxyMode,
		ProxySecret:      c.ProxySecret,
		PublicTenantID:   c.PublicTenantID,
	}
}
