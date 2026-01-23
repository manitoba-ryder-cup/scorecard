package main

import (
	"github.com/urfave/cli/v2"
)

// Common flags that can be reused across commands
var (
	// DebugFlag enables debug logging (global flag)
	DebugFlag = &cli.BoolFlag{
		Name:        "debug",
		Usage:       "Enable debug logging",
		EnvVars:     []string{"DEBUG"},
		Destination: &config.Debug,
	}

	// DatabaseURLFlag defines the PostgreSQL connection URL (global flag)
	DatabaseURLFlag = &cli.StringFlag{
		Name:        "database-url",
		Aliases:     []string{"u"},
		Usage:       "PostgreSQL connection URL",
		Value:       "postgres://postgres:password@localhost:5432/scorecard?sslmode=disable",
		EnvVars:     []string{"DATABASE_URL"},
		Destination: &config.DatabaseURL,
	}

	// HTTPAddressFlag defines the HTTP server listen address
	HTTPAddressFlag = &cli.StringFlag{
		Name:        "http-address",
		Aliases:     []string{"a"},
		Usage:       "HTTP address to listen on",
		Value:       ":5000",
		EnvVars:     []string{"HTTP_ADDRESS"},
		Destination: &config.HTTPAddress,
	}

	// JWTPublicKeyFlag defines the path to the JWT public key
	JWTPublicKeyFlag = &cli.StringFlag{
		Name:        "jwt-public-key",
		Aliases:     []string{"p"},
		Usage:       "Path to JWT public key file (PEM format)",
		Required:    true,
		EnvVars:     []string{"JWT_PUBLIC_KEY_PATH"},
		Destination: &config.JWTPublicKeyPath,
	}

	// EnvironmentFlag defines the deployment environment
	EnvironmentFlag = &cli.StringFlag{
		Name:        "environment",
		Aliases:     []string{"e"},
		Usage:       "Environment (development, staging, production)",
		Value:       "development",
		EnvVars:     []string{"ENVIRONMENT"},
		Destination: &config.Environment,
	}
)
