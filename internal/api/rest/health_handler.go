package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// HandleHealth returns the service health status
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := sdk.HealthResponse{
		Status: "OK",
	}

	respondJSON(w, http.StatusOK, response)
}
