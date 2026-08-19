package test

import (
	"encoding/json"
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

	// A 200 alone passes on an empty body, so the probe would look healthy while
	// telling a caller nothing.
	var body sdk.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "OK" {
		t.Errorf("status = %q, want %q", body.Status, "OK")
	}
}
