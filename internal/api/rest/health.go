package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// An unreachable database yields 503 rather than 500 so a load balancer routes around
// this instance instead of treating the request as at fault.
func (r *Router) HandleHealth(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
	defer cancel()

	if err := r.DB.Health(ctx); err != nil {
		respondError(ctx, w, http.StatusServiceUnavailable, "database unavailable", err)
		return
	}

	respondJSON(w, http.StatusOK, sdk.HealthResponse{Status: "OK"})
}
