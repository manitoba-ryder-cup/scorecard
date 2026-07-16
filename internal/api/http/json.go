package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// respondJSON sends a JSON response
func respondJSON(writer http.ResponseWriter, status int, data any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err, "status", status)
	}
}

// respondError sends an error response using the SDK's error envelope.
func respondError(writer http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		slog.Error("API error", "message", message, "error", err, "status", status)
	}
	respondJSON(writer, status, sdk.ErrorResponse{Error: message})
}
