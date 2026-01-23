package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// respondJSON sends a JSON response
func respondJSON(writer http.ResponseWriter, status int, data any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err, "status", status)
	}
}

// respondError sends an error response
func respondError(writer http.ResponseWriter, status int, message string, err error) {
	// Log internal error for debugging
	if err != nil {
		slog.Error("API error", "message", message, "error", err, "status", status)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if encodeErr := json.NewEncoder(writer).Encode(map[string]string{"error": message}); encodeErr != nil {
		slog.Error("Failed to encode error response", "error", encodeErr, "status", status)
	}
}
