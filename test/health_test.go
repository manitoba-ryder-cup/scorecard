package test

import (
	"net/http"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// TestHealthReportsOK checks that /health returns 200 when the database is reachable
// (the readiness probe the deployment relies on).
func TestHealthReportsOK(t *testing.T) {
	t.Parallel()
	resp, err := http.Get(util.LoadConfig().BaseURL + sdk.RouteHealth)
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}
