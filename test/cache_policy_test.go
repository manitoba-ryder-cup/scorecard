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

// The three cache tiers are unit-tested in internal/api/rest/cache_test.go, but only in
// isolation: nothing there checks that a real cup reaches the phase it is in. Getting that
// wrong is invisible in the body — a frozen leaderboard is served with the same 200 and the
// same JSON as a correct one, just a stale copy of it, and only during play.
//
// These read the fixture anonymously, which is both the path that gets cached and the
// only one that can be: cacheableRead marks a response only when the request carries no
// Authorization header.

// publicFixture seeds a singles match under the public tenant so an anonymous read can
// see it, and returns the fixture with a client authenticated for that same tenant.
func publicFixture(t *testing.T) (*sdk.Client, *util.Fixture) {
	t.Helper()
	cfg := util.LoadConfig()
	ctx := context.Background()

	conn, err := util.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	fix, err := util.SeedSinglesMatchFor(ctx, conn, util.PublicTenantID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	client := sdk.NewClient(cfg.BaseURL)
	client.SetToken(testjwt.MintAccessToken(t, fix.TenantID, uuid.New(), writeScopes...))
	return client, fix
}

func cacheControl(t *testing.T, path, token string) string {
	t.Helper()
	status, header := request.Headers(t, path, token)
	if status != http.StatusOK {
		t.Fatalf("GET %s: want 200 before judging its caching, got %d", path, status)
	}
	return header.Get("Cache-Control")
}

func TestLiveTournamentReadsAreNotCached(t *testing.T) {
	t.Parallel()
	client, fix := publicFixture(t)
	playHole(t, client, fix, 1, 4, 5) // one hole in: the cup is being played

	// A spectator is polling both of these every twenty seconds. Caching either would
	// answer two polls in three from a copy taken before the score they are waiting for.
	for _, path := range []string{
		strings.Replace(sdk.RouteV1TournamentResults, "{id}", fix.TournamentID.String(), 1),
		strings.Replace(sdk.RouteV1TournamentTeams, "{id}", fix.TournamentID.String(), 1),
		strings.Replace(sdk.RouteV1Tournament, "{id}", fix.TournamentID.String(), 1),
	} {
		if got := cacheControl(t, path, ""); got != "no-store" {
			t.Errorf("%s during play: want no-store, got %q", path, got)
		}
	}
}

func TestSettledTournamentReadsAreCachedForADay(t *testing.T) {
	t.Parallel()
	client, fix := publicFixture(t)
	closeOutRedWin(t, client, fix)

	for _, path := range []string{
		strings.Replace(sdk.RouteV1TournamentResults, "{id}", fix.TournamentID.String(), 1),
		strings.Replace(sdk.RouteV1TournamentTeams, "{id}", fix.TournamentID.String(), 1),
		strings.Replace(sdk.RouteV1Tournament, "{id}", fix.TournamentID.String(), 1),
	} {
		if got := cacheControl(t, path, ""); got != "public, max-age=86400" {
			t.Errorf("%s once finished: want the settled tier, got %q", path, got)
		}
	}
}

// Everything without a live/settled opinion takes the default minute: short enough not to
// strand a correction, long enough to absorb a burst.
func TestOrdinaryReadsTakeTheDefaultTier(t *testing.T) {
	t.Parallel()
	if got := cacheControl(t, sdk.RouteV1Players, ""); got != "public, max-age=60" {
		t.Errorf("%s: want the default tier, got %q", sdk.RouteV1Players, got)
	}
}

func TestScheduledButUnplayedReadsTakeTheDefaultTier(t *testing.T) {
	t.Parallel()
	_, fix := publicFixture(t)

	for _, path := range []string{
		strings.Replace(sdk.RouteV1TournamentResults, "{id}", fix.TournamentID.String(), 1),
		strings.Replace(sdk.RouteV1TournamentTeams, "{id}", fix.TournamentID.String(), 1),
		strings.Replace(sdk.RouteV1Tournament, "{id}", fix.TournamentID.String(), 1),
	} {
		if got := cacheControl(t, path, ""); got != "public, max-age=60" {
			t.Errorf("%s before play: want the default tier, got %q", path, got)
		}
	}
}

// Only scorers hold a token, and a scorer submits a hole then immediately refetches.
// Serving them their own pre-submission data is the one staleness that would genuinely
// mislead, so an authenticated read is never cacheable — including on the routes whose
// anonymous form is.
func TestAuthenticatedReadsAreNeverCacheable(t *testing.T) {
	t.Parallel()
	client, fix := publicFixture(t)
	closeOutRedWin(t, client, fix) // the settled tier, which is the one worth defeating

	token := testjwt.MintAccessToken(t, util.PublicTenantID, uuid.New(), writeScopes...)
	for _, path := range []string{
		sdk.RouteV1Players,
		strings.Replace(sdk.RouteV1TournamentResults, "{id}", fix.TournamentID.String(), 1),
	} {
		if got := cacheControl(t, path, token); got != "no-store" {
			t.Errorf("%s with a token: want no-store, got %q", path, got)
		}
	}
}

// The two endpoints derive the cup's phase from different rows, and this is where that
// showed: /results reads live scores while /teams and the record read match_results.
// Removing a participant cascades their scores away but leaves the materialized result
// behind, so the two disagreed about the same cup at the same instant.
func TestPhaseAgreesAcrossReadsAfterAParticipantIsRemoved(t *testing.T) {
	t.Parallel()
	client, fix := publicFixture(t)
	closeOutRedWin(t, client, fix)

	if err := client.RemoveParticipant(context.Background(), fix.MatchID, fix.RedPlayer); err != nil {
		t.Fatalf("remove participant: %v", err)
	}

	results := strings.Replace(sdk.RouteV1TournamentResults, "{id}", fix.TournamentID.String(), 1)
	record := strings.Replace(sdk.RouteV1Tournament, "{id}", fix.TournamentID.String(), 1)
	if got, want := cacheControl(t, results, ""), cacheControl(t, record, ""); got != want {
		t.Errorf("one cup, two phases: /results %q, the record %q", got, want)
	}
}
