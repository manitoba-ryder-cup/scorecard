package isolation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	testjwt "github.com/manitoba-ryder-cup/scorecard/test/_util/jwt"
)

// writeScopes covers every scope, so a cross-tenant write is rejected for crossing the
// boundary and not merely for lacking a scope.
var writeScopes = []string{
	sdk.ScopeTournamentsWrite,
	sdk.ScopePlayersWrite,
	sdk.ScopeScoresWrite,
	sdk.ScopeCoursesWrite,
}

// TestMain preflights the infrastructure so `go test ./test/...` without a running
// stack skips with a clear hint instead of failing on connection errors.
func TestMain(m *testing.M) {
	cfg := util.LoadConfig()
	ctx := context.Background()

	conn, err := util.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("SKIP: cannot reach test database (%v). Run `make test-setup` first.\n", err)
		os.Exit(0)
	}
	_ = conn.Close(ctx)

	if _, err := http.Get(cfg.BaseURL + sdk.RouteHealth); err != nil {
		fmt.Printf("SKIP: cannot reach scorecard at %s (%v). Run `make test-setup` first.\n", cfg.BaseURL, err)
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// tenantClient returns an SDK client authenticated for tenantID with full write scopes.
func tenantClient(t *testing.T, tenantID uuid.UUID) *sdk.Client {
	t.Helper()
	client := sdk.NewClient(util.LoadConfig().BaseURL)
	client.SetToken(testjwt.MintAccessToken(t, tenantID, uuid.New(), writeScopes...))
	return client
}

// requireNotFound insists on 404, not 403: a tenant must not be able to distinguish
// "belongs to someone else" from "does not exist".
func requireNotFound(t *testing.T, err error, what string) {
	t.Helper()
	var apiErr *sdk.APIError
	if err == nil {
		t.Fatalf("%s: cross-tenant access succeeded; want 404", what)
	}
	if !errors.As(err, &apiErr) {
		t.Fatalf("%s: want *sdk.APIError, got %v", what, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("%s: want 404, got %d (%s)", what, apiErr.StatusCode, apiErr.Message)
	}
}

// requireRejected only insists the write failed — the status varies by endpoint (a
// missing parent is 404, a foreign reference trips an FK as 400).
func requireRejected(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: cross-tenant write succeeded; want rejection", what)
	}
	var apiErr *sdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("%s: want *sdk.APIError, got %v", what, err)
	}
	if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
		t.Fatalf("%s: want a 4xx rejection, got %d (%s)", what, apiErr.StatusCode, apiErr.Message)
	}
}
