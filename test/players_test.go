package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

func TestCreatePlayerAndReadBack(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()

	email := "dustin@example.com"
	created, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{
		FirstName: "Dustin", LastName: "Johnson", Email: &email,
	})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	if created.ID == 0 || created.FirstName != "Dustin" || created.LastName != "Johnson" {
		t.Fatalf("unexpected player: %+v", created)
	}

	// Reads back as a profile with an empty record (no matches played yet).
	got, err := client.GetPlayer(ctx, created.ID)
	if err != nil {
		t.Fatalf("get player: %v", err)
	}
	if got.ID != created.ID || got.Email == nil || *got.Email != email {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.Record.Wins != 0 || got.Record.Losses != 0 || got.Record.Ties != 0 {
		t.Errorf("new player should have an empty record, got %+v", got.Record)
	}

	// Appears in the roster listing.
	players, err := client.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("list players: %v", err)
	}
	found := false
	for _, p := range players {
		if p.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Error("created player not in roster listing")
	}
}

func TestCreatePlayerRosterOnly(t *testing.T) {
	client := freshClient(t)

	// No email, no user_id — a roster-only player is valid.
	created, err := client.CreatePlayer(context.Background(), sdk.CreatePlayerRequest{
		FirstName: "Roster", LastName: "Only",
	})
	if err != nil {
		t.Fatalf("create roster-only player: %v", err)
	}
	if created.Email != nil || created.UserID != nil {
		t.Errorf("want nil email/user_id, got %+v", created)
	}
}

func TestCreatePlayerDuplicateEmailConflicts(t *testing.T) {
	client := freshClient(t)
	ctx := context.Background()
	email := "dup@example.com"

	if _, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "First", LastName: "Player", Email: &email}); err != nil {
		t.Fatalf("create first: %v", err)
	}

	// Same email under the same tenant collides with UNIQUE(tenant_id, email) -> 409.
	_, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Second", LastName: "Player", Email: &email})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 APIError, got %v", err)
	}
}

func TestGetNonexistentPlayerReturns404(t *testing.T) {
	client := freshClient(t)

	_, err := client.GetPlayer(context.Background(), 999999)
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

// Raw request (bypassing the SDK client's validation) confirms the server rejects a
// nameless player too.
func TestCreatePlayerEmptyNameRejectedByServer(t *testing.T) {
	body := `{"first_name":"","last_name":"Johnson"}`
	status, _ := request.Raw(t, http.MethodPost, sdk.RouteV1Players, body, freshToken(t))
	if status != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", status)
	}
}
