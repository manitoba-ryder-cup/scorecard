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
