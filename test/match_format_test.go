package test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
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

// Fatal on a name nothing seeded: a zero id would fail later as something else.
func formatNamed(t *testing.T, client *sdk.Client, name string) uuid.UUID {
	t.Helper()
	id, ok := formatsByName(t, client)[name]
	if !ok {
		t.Fatalf("no %s format seeded", name)
	}
	return id
}

// matchInFormat creates a match in the named format. A format is chosen once, so a test that
// needs a different one asks for another match rather than editing this one.
func matchInFormat(t *testing.T, client *sdk.Client, fix *util.Fixture, name string) uuid.UUID {
	t.Helper()
	m, err := client.CreateMatch(context.Background(), fix.TournamentID, sdk.CreateMatchRequest{
		CourseID:      fix.CourseID,
		TeeColorID:    fix.TeeColorID,
		MatchFormatID: formatNamed(t, client, name),
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
	otherFormat := formatNamed(t, client, "Alt Shot")

	// Sent raw, not through the SDK: the point is a caller that puts a field the contract
	// does not carry, which the typed request cannot express.
	path := strings.Replace(sdk.RouteV1Match, "{id}", fix.MatchID.String(), 1)
	token := testjwt.MintAccessToken(t, fix.TenantID, uuid.New(), writeScopes...)
	status, _ := request.Raw(t, http.MethodPut, path, `{"match_format_id":"`+otherFormat.String()+`"}`, token)

	if status != http.StatusBadRequest {
		t.Errorf("want 400 for a field the contract does not carry, got %d", status)
	}
	if now := theMatch(t, client, fix).MatchFormatID; now != before {
		t.Errorf("the format changed to %s", now)
	}
}
