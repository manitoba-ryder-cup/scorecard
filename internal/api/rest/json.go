package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
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

// respondError sends an error response using the SDK's error envelope. Server faults
// (5xx) log at Error; client errors (4xx) log at Warn, so a bad request or a missing
// resource doesn't pollute the error stream.
func respondError(writer http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		if status >= http.StatusInternalServerError {
			slog.Error("API error", "message", message, "error", err, "status", status)
		} else {
			slog.Warn("API client error", "message", message, "error", err, "status", status)
		}
	}
	respondJSON(writer, status, sdk.ErrorResponse{Error: message})
}

// respondDomainError maps a domain sentinel to the right HTTP status: not found -> 404,
// invalid input -> 400, conflict -> 409, anything else -> 500. Keeps handlers from
// re-deriving the mapping and keeps status semantics in one place.
func respondDomainError(writer http.ResponseWriter, message string, err error) {
	switch {
	case errors.Is(err, golf.ErrNotFound):
		respondError(writer, http.StatusNotFound, message, err)
	case errors.Is(err, golf.ErrInvalidInput):
		respondError(writer, http.StatusBadRequest, message, err)
	case errors.Is(err, golf.ErrConflict):
		respondError(writer, http.StatusConflict, message, err)
	default:
		respondError(writer, http.StatusInternalServerError, message, err)
	}
}
