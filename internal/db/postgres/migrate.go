package postgres

import (
	"embed"
	"strings"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" driver golang-migrate opens from the URL scheme
	"github.com/travisbale/knowhere/db"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrateURL(databaseURL string) string {
	for _, scheme := range []string{"postgresql://", "postgres://"} {
		if rest, ok := strings.CutPrefix(databaseURL, scheme); ok {
			return "pgx5://" + rest
		}
	}
	return databaseURL
}

// MigrateUp applies all pending migrations using knowhere's migration helper
func MigrateUp(databaseURL string) error {
	return db.MigrateUp(migrationsFS, "migrations", migrateURL(databaseURL))
}

// MigrateDown rolls back the last migration using knowhere's migration helper
func MigrateDown(databaseURL string) error {
	return db.MigrateDown(migrationsFS, "migrations", migrateURL(databaseURL))
}

// MigrateVersion returns the current migration version using knowhere's migration helper
func MigrateVersion(databaseURL string) (version uint, dirty bool, err error) {
	return db.MigrateVersion(migrationsFS, "migrations", migrateURL(databaseURL))
}
