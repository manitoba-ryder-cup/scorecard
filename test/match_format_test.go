package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
)

// theMatch reads the fixture's match back off the tournament listing; the SDK has no
// single-match read.
func theMatch(t *testing.T, client *sdk.Client, fix *util.Fixture) sdk.Match {
	t.Helper()
	matches, err := client.ListMatches(context.Background(), fix.TournamentID)
	if err != nil {
		t.Fatalf("list matches: %v", err)
	}
	for _, m := range matches {
		if m.ID == fix.MatchID {
			return m
		}
	}
	t.Fatalf("match %s not in the tournament listing", fix.MatchID)
	return sdk.Match{}
}

func formatNamed(t *testing.T, client *sdk.Client, name string) sdk.MatchFormat {
	t.Helper()
	formats, err := client.ListMatchFormats(context.Background())
	if err != nil {
		t.Fatalf("list formats: %v", err)
	}
	for _, f := range formats {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no %s format seeded", name)
	return sdk.MatchFormat{}
}

// matchInFormat creates a match in the named format. A format is chosen once, so a test that
// needs a different one asks for another match rather than editing this one.
func matchInFormat(t *testing.T, client *sdk.Client, fix *util.Fixture, name string) uuid.UUID {
	t.Helper()
	m, err := client.CreateMatch(context.Background(), fix.TournamentID, sdk.CreateMatchRequest{
		CourseID:      fix.CourseID,
		TeeColorID:    fix.TeeColorID,
		MatchFormatID: formatNamed(t, client, name).ID,
		TeeTime:       theMatch(t, client, fix).TeeTime,
	})
	if err != nil {
		t.Fatalf("create a %s match: %v", name, err)
	}
	return m.ID
}

// The format decides how many play a side and whether a hole is recorded per player, so
// changing it reinterprets a match rather than adjusting it. It is not on the update contract,
// and a caller that sends it anyway is told so rather than having it quietly ignored.
func TestAMatchFormatCannotBeChanged(t *testing.T) {
	t.Parallel()
	client, fix := authedClient(t)
	before := theMatch(t, client, fix).MatchFormatID
	other := formatNamed(t, client, "Alt Shot")

	body, _ := json.Marshal(map[string]string{"match_format_id": other.ID.String()})
	req, err := http.NewRequest(http.MethodPut,
		util.LoadConfig().BaseURL+"/v1/matches/"+fix.MatchID.String(), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testjwt.MintAccessToken(t, fix.TenantID, uuid.New(), writeScopes...))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400 for a field the contract does not carry, got %d", resp.StatusCode)
	}
	if now := theMatch(t, client, fix).MatchFormatID; now != before {
		t.Errorf("the format changed to %s", now)
	}
}
