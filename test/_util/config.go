// Package util holds shared helpers for the scorecard integration suite: test
// configuration, a database-seeding fixture, and (in subpackages) JWT minting.
package util

import (
	"os"
	"path/filepath"
	"runtime"
)

// Config holds the integration suite's connection details. Defaults point at the
// local docker-compose infrastructure (test/docker-compose.yml).
type Config struct {
	BaseURL           string // scorecard API base URL
	DatabaseURL       string // superuser connection used to seed fixtures (bypasses RLS)
	JWTPrivateKeyPath string // private key the test issuer signs tokens with
}

// LoadConfig reads configuration from the environment, falling back to the
// docker-compose defaults.
func LoadConfig() *Config {
	return &Config{
		BaseURL:           getEnv("SCORECARD_BASE_URL", "http://localhost:5000"),
		DatabaseURL:       getEnv("TEST_DATABASE_URL", "postgres://superuser:superuser@localhost:5433/scorecard?sslmode=disable"),
		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", privateKeyPath()),
	}
}

// privateKeyPath resolves test/keys/private-key.pem relative to this source file,
// so tests run from any working directory.
func privateKeyPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "keys", "private-key.pem")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
