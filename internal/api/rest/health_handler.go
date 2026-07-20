package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// HandleHealth returns a handler that reports service health, verifying database
// readiness with a short timeout. A DB that can't be reached yields 503 so a load
// balancer can route around an unready instance.
func HandleHealth(db HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Health(ctx); err != nil {
			respondError(ctx, w, http.StatusServiceUnavailable, "database unavailable", err)
			return
		}
		respondJSON(w, http.StatusOK, sdk.HealthResponse{Status: "OK"})
	}
}
