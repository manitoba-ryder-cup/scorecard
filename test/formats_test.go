package test

import (
	"context"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// TestListMatchFormatsIsPublicAndSeeded confirms match formats are global seeded
// reference data: readable with no token (no tenant) and populated by the migration.
func TestListMatchFormatsIsPublicAndSeeded(t *testing.T) {
	t.Parallel()
	client := sdk.NewClient(util.LoadConfig().BaseURL) // no token

	formats, err := client.ListMatchFormats(context.Background())
	if err != nil {
		t.Fatalf("list match formats: %v", err)
	}

	names := make(map[string]bool, len(formats))
	for _, f := range formats {
		names[f.Name] = true
	}
	for _, want := range []string{"Singles", "Fourball", "Alt Shot", "Scramble", "Scotch"} {
		if !names[want] {
			t.Errorf("missing seeded format %q (got %+v)", want, formats)
		}
	}
}

// The web client looks a format up by name, so order is cosmetic — but it comes from
// gen_random_uuid() PKs today, which shuffles differently in every database.
func TestListMatchFormatsIsSortedByName(t *testing.T) {
	t.Parallel()

	formats, err := sdk.NewClient(util.LoadConfig().BaseURL).ListMatchFormats(context.Background())
	if err != nil {
		t.Fatalf("list match formats: %v", err)
	}

	for i := 1; i < len(formats); i++ {
		if formats[i-1].Name > formats[i].Name {
			names := make([]string, len(formats))
			for j, f := range formats {
				names[j] = f.Name
			}
			t.Fatalf("formats are not sorted by name: %v", names)
		}
	}
}

// Pinned per format: one wrong pair is a match that fields the wrong number of players, or
// reads the wrong number of scores off the ones it has.
func TestEveryFormatCarriesItsRules(t *testing.T) {
	t.Parallel()
	client := sdk.NewClient(util.LoadConfig().BaseURL) // no token

	formats, err := client.ListMatchFormats(context.Background())
	if err != nil {
		t.Fatalf("list formats: %v", err)
	}

	want := map[string]struct {
		perSide         int32
		scoresPerPlayer bool
	}{
		"Singles":  {1, true},
		"Fourball": {2, true},
		"Alt Shot": {2, false},
		"Scramble": {2, false},
		"Scotch":   {2, false},
	}
	if len(formats) != len(want) {
		t.Fatalf("want %d seeded formats, got %d", len(want), len(formats))
	}
	for _, f := range formats {
		w, ok := want[f.Name]
		if !ok {
			t.Errorf("unexpected format %q", f.Name)
			continue
		}
		if f.PlayersPerSide != w.perSide {
			t.Errorf("%s: want %d a side, got %d", f.Name, w.perSide, f.PlayersPerSide)
		}
		if f.ScoresPerPlayer != w.scoresPerPlayer {
			t.Errorf("%s: want scores_per_player %v, got %v", f.Name, w.scoresPerPlayer, f.ScoresPerPlayer)
		}
	}
}
