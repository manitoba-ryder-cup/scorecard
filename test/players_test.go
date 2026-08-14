package test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	"github.com/manitoba-ryder-cup/scorecard/test/_util/request"
)

func TestCreatePlayerAndReadBack(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	email := "dustin@example.com"
	created, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{
		FirstName: "Dustin", LastName: "Johnson", Email: &email,
	})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	if created.ID == uuid.Nil || created.FirstName != "Dustin" || created.LastName != "Johnson" {
		t.Fatalf("unexpected player: %+v", created)
	}

	// Reads back as a profile with an empty record (no matches played yet).
	got, err := client.GetPlayer(ctx, created.ID)
	if err != nil {
		t.Fatalf("get player: %v", err)
	}
	// Email is write-only (see TestPlayerEmailIsNeverReturned), so identity round-trips
	// on the name.
	if got.ID != created.ID || got.FirstName != "Dustin" || got.LastName != "Johnson" {
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
			// The listing carries each player's record (the players page renders it).
			if p.Record.Wins != 0 || p.Record.Losses != 0 || p.Record.Ties != 0 {
				t.Errorf("new player should list an empty record, got %+v", p.Record)
			}
		}
	}
	if !found {
		t.Error("created player not in roster listing")
	}
}

// A photo is the player's, not a tournament's, so it is set and cleared here rather than
// on a roster entry.
func TestUpdatePlayerPhoto(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	ctx := context.Background()

	created, err := client.CreatePlayer(ctx, sdk.CreatePlayerRequest{FirstName: "Nigel", LastName: "Milnes"})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	photo := "/img/players/nigel.webp"
	updated, err := client.UpdatePlayer(ctx, created.ID, sdk.UpdatePlayerRequest{PhotoPath: &photo})
	if err != nil {
		t.Fatalf("set photo: %v", err)
	}
	if updated.PhotoPath != photo {
		t.Fatalf("want photo %q, got %q", photo, updated.PhotoPath)
	}
	// The name was not in the body and must not have been blanked by the write.
	if updated.FirstName != "Nigel" || updated.LastName != "Milnes" {
		t.Fatalf("an omitted field was overwritten: %+v", updated)
	}

	cleared := ""
	after, err := client.UpdatePlayer(ctx, created.ID, sdk.UpdatePlayerRequest{PhotoPath: &cleared})
	if err != nil {
		t.Fatalf("clear photo: %v", err)
	}
	if after.PhotoPath != "" {
		t.Errorf("want the photo cleared, got %q", after.PhotoPath)
	}

	got, err := client.GetPlayer(ctx, created.ID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got.PhotoPath != "" {
		t.Errorf("cleared photo did not persist: %q", got.PhotoPath)
	}
}

func TestUpdateNonexistentPlayerReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)
	name := "Ghost"
	_, err := client.UpdatePlayer(context.Background(), uuid.New(), sdk.UpdatePlayerRequest{FirstName: &name})
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %v", err)
	}
}

func TestCreatePlayerRosterOnly(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	// No email, no user_id — a roster-only player is valid.
	created, err := client.CreatePlayer(context.Background(), sdk.CreatePlayerRequest{
		FirstName: "Roster", LastName: "Only",
	})
	if err != nil {
		t.Fatalf("create roster-only player: %v", err)
	}
	if created.UserID != nil {
		t.Errorf("want nil user_id, got %+v", created)
	}
}

func TestCreatePlayerDuplicateEmailConflicts(t *testing.T) {
	t.Parallel()
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

// TestAnonymousReadUsesPublicTenant confirms a request with no token can read data
// belonging to the configured public tenant — the spectator path for a public site.
func TestAnonymousReadUsesPublicTenant(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := util.LoadConfig()

	conn, err := util.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	// Seed a player directly under the public tenant (not via a tokened request).
	id, err := util.SeedPlayer(ctx, conn, util.PublicTenantID, "Public", "Viewer")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// A client with no token reads it — the server resolves the public tenant.
	client := sdk.NewClient(cfg.BaseURL)
	got, err := client.GetPlayer(ctx, id)
	if err != nil {
		t.Fatalf("anonymous get player: %v", err)
	}
	if got.ID != id || got.LastName != "Viewer" {
		t.Fatalf("unexpected player: %+v", got)
	}
}

func TestGetNonexistentPlayerReturns404(t *testing.T) {
	t.Parallel()
	client := freshClient(t)

	_, err := client.GetPlayer(context.Background(), uuid.New())
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 APIError, got %v", err)
	}
}

// Raw request (bypassing the SDK client's validation) confirms the server rejects a
// nameless player too.
func TestCreatePlayerEmptyNameRejectedByServer(t *testing.T) {
	t.Parallel()
	body := `{"first_name":"","last_name":"Johnson"}`
	status, _ := request.Raw(t, http.MethodPost, sdk.RouteV1Players, body, freshToken(t))
	if status != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", status)
	}
}
