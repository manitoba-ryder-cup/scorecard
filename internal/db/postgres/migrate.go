package postgres

import (
	"embed"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers the "postgres" driver golang-migrate opens from the DATABASE_URL scheme
	"github.com/travisbale/knowhere/db"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateUp applies all pending migrations using knowhere's migration helper
func MigrateUp(databaseURL string) error {
	return db.MigrateUp(migrationsFS, "migrations", databaseURL)
}

// MigrateDown rolls back the last migration using knowhere's migration helper
func MigrateDown(databaseURL string) error {
	return db.MigrateDown(migrationsFS, "migrations", databaseURL)
}

// MigrateVersion returns the current migration version using knowhere's migration helper
func MigrateVersion(databaseURL string) (version uint, dirty bool, err error) {
	return db.MigrateVersion(migrationsFS, "migrations", databaseURL)
}
