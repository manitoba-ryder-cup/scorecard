package postgres

import "testing"

// The scheme selects the driver, so a URL left saying "postgres" finds none registered and
// migrations fail at startup.
func TestMigrateURLRetargetsThePgxDriver(t *testing.T) {
	cases := map[string]string{
		"postgres://u:p@localhost:5432/db?sslmode=disable":   "pgx5://u:p@localhost:5432/db?sslmode=disable",
		"postgresql://u:p@localhost:5432/db?sslmode=disable": "pgx5://u:p@localhost:5432/db?sslmode=disable",
		// Already pointed at the driver, or something we do not recognise: left alone.
		"pgx5://u:p@localhost/db": "pgx5://u:p@localhost/db",
		"":                        "",
	}
	for in, want := range cases {
		if got := migrateURL(in); got != want {
			t.Errorf("migrateURL(%q) = %q, want %q", in, got, want)
		}
	}
}
