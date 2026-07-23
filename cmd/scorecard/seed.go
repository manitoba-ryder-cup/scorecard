package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/app"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
	"github.com/travisbale/knowhere/identity"
	"github.com/urfave/cli/v2"
)

var seedTournamentCmd = &cli.Command{
	Name:      "seed-tournament",
	Usage:     "Create a tournament, its roster (with captains), and the match schedule from JSON",
	ArgsUsage: " ",
	Description: "Reads a setup JSON (from --file or stdin) and creates the tournament, its two teams, " +
		"the entered roster with each side's captain, and the matches for each format. It does not draft " +
		"the field or assign match participants (those happen live at the event). The course, tee color, " +
		"and formats are referenced by name and must already exist.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "tenant-id",
			Usage:    "Tenant the tournament belongs to",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "file",
			Aliases: []string{"f"},
			Usage:   "Setup JSON file (default: read stdin)",
		},
	},
	Action: func(c *cli.Context) error {
		tenantID, err := uuid.Parse(c.String("tenant-id"))
		if err != nil {
			return fmt.Errorf("invalid tenant-id: %w", err)
		}

		in, err := readSeedInput(c.String("file"))
		if err != nil {
			return err
		}

		db, err := postgres.NewDB(c.Context, config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connecting to database: %w", err)
		}
		defer db.Close()

		ctx := identity.WithTenant(c.Context, tenantID)
		summary, err := app.SeedTournament(ctx, app.NewServices(db), in)
		if err != nil {
			return err
		}

		fmt.Printf("Created tournament %s: %d players entered, %d matches (roster undrafted, no participants).\n",
			summary.TournamentID, summary.PlayersEntered, summary.Matches)
		return nil
	},
}

// readSeedInput decodes the setup JSON from a file, or stdin when no path is given.
// Unknown fields are rejected so a typo'd key fails loudly rather than being ignored.
func readSeedInput(path string) (*app.SeedInput, error) {
	r := io.Reader(os.Stdin)
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", path, err)
		}
		defer f.Close()
		r = f
	}

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var in app.SeedInput
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("parsing setup JSON: %w", err)
	}
	return &in, nil
}
