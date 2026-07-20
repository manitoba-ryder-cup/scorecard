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
	for _, want := range []string{"Singles", "Fourball", "Alternate Shot", "Scramble", "Modified Scotch"} {
		if !names[want] {
			t.Errorf("missing seeded format %q (got %+v)", want, formats)
		}
	}
}
