package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// maxRequestBody caps decoded request bodies to guard against a client streaming an
// unbounded payload into memory.
const maxRequestBody = 1 << 20 // 1 MiB

// validatable is implemented by the SDK request types (Validate runs client-side too).
type validatable interface {
	Validate(ctx context.Context) error
}

// decodeAndValidate reads a size-limited JSON body into a T and validates its shape,
// writing a 400 and returning ok=false on any failure. It collapses the identical
// decode -> validate -> respond preamble every write handler shared.
func decodeAndValidate[T validatable](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid request body", err)
		return req, false
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, err.Error(), nil)
		return req, false
	}
	return req, true
}

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
// resource doesn't pollute the error stream. Logging uses the *Context variants so the
// request's tenant/actor/request-id (injected by identity.LogHandler) ride along.
func respondError(ctx context.Context, writer http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		if status >= http.StatusInternalServerError {
			slog.ErrorContext(ctx, "API error", "message", message, "error", err, "status", status)
		} else {
			slog.WarnContext(ctx, "API client error", "message", message, "error", err, "status", status)
		}
	}
	respondJSON(writer, status, sdk.ErrorResponse{Error: message})
}

// respondDomainError maps a domain sentinel to the right HTTP status: not found -> 404,
// invalid input -> 400, conflict -> 409, anything else -> 500. Keeps handlers from
// re-deriving the mapping and keeps status semantics in one place.
func respondDomainError(ctx context.Context, writer http.ResponseWriter, message string, err error) {
	switch {
	case errors.Is(err, golf.ErrNotFound):
		respondError(ctx, writer, http.StatusNotFound, message, err)
	case errors.Is(err, golf.ErrInvalidInput):
		respondError(ctx, writer, http.StatusBadRequest, message, err)
	case errors.Is(err, golf.ErrConflict):
		respondError(ctx, writer, http.StatusConflict, message, err)
	default:
		respondError(ctx, writer, http.StatusInternalServerError, message, err)
	}
}
