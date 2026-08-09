package app

import (
	"strings"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// These are the errors that used to surface only after the tournament, the roster and
// both captains had been written. They cost nothing now, so the assertion that matters
// is that they are reached at all.

func TestPlanRoster(t *testing.T) {
	t.Run("keys the roster by lowercased email", func(t *testing.T) {
		roster, err := planRoster([]SeedPlayer{{FirstName: "Jon", LastName: "Ray", Email: "Jon.Ray@Example.com"}})
		if err != nil {
			t.Fatalf("planRoster: %v", err)
		}
		if _, ok := roster["jon.ray@example.com"]; !ok {
			t.Errorf("want the address lowercased, got keys %v", keys(roster))
		}
	})

	t.Run("rejects a player with no email", func(t *testing.T) {
		_, err := planRoster([]SeedPlayer{{FirstName: "Jon", LastName: "Ray"}})
		if err == nil || !strings.Contains(err.Error(), "no email") {
			t.Errorf("want a missing-email error naming the player, got %v", err)
		}
	})

	t.Run("rejects whitespace masquerading as an email", func(t *testing.T) {
		if _, err := planRoster([]SeedPlayer{{FirstName: "Jon", Email: "   "}}); err == nil {
			t.Error("want an error, got nil")
		}
	})

	// Entering the same person twice would give them two roster rows under one identity.
	t.Run("rejects a repeated email", func(t *testing.T) {
		_, err := planRoster([]SeedPlayer{
			{FirstName: "Jon", Email: "jon@example.com"},
			{FirstName: "Jonathan", Email: "JON@example.com"},
		})
		if err == nil || !strings.Contains(err.Error(), "twice") {
			t.Errorf("want a duplicate error, got %v", err)
		}
	})
}

func TestPlanCaptains(t *testing.T) {
	roster := map[string]SeedPlayer{"jon@example.com": {FirstName: "Jon", Email: "jon@example.com"}}

	t.Run("accepts a captain who is on the roster", func(t *testing.T) {
		got, err := planCaptains(map[string]string{sdk.TeamColorRed: "Jon@Example.com"}, roster)
		if err != nil {
			t.Fatalf("planCaptains: %v", err)
		}
		if got[sdk.TeamColorRed] != "jon@example.com" {
			t.Errorf("want the lookup key normalised, got %q", got[sdk.TeamColorRed])
		}
	})

	t.Run("rejects a captain who is not entered", func(t *testing.T) {
		_, err := planCaptains(map[string]string{sdk.TeamColorBlue: "nobody@example.com"}, roster)
		if err == nil || !strings.Contains(err.Error(), "not in the roster") {
			t.Errorf("want a not-in-roster error, got %v", err)
		}
	})

	// A tournament only ever has these two sides, so a third colour is a typo that would
	// otherwise be found after the roster was already committed.
	t.Run("rejects a colour that is not one of the two sides", func(t *testing.T) {
		_, err := planCaptains(map[string]string{"Green": "jon@example.com"}, roster)
		if err == nil || !strings.Contains(err.Error(), "Green") {
			t.Errorf("want the bad colour named, got %v", err)
		}
	})
}

func keys(m map[string]SeedPlayer) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
