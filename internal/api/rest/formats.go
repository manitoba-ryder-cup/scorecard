package rest

import (
	"context"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type FormatService interface {
	ListFormats(ctx context.Context) ([]golf.MatchFormat, error)
}

type FormatsHandler struct {
	formatService FormatService
}

func NewFormatsHandler(formatService FormatService) *FormatsHandler {
	return &FormatsHandler{formatService: formatService}
}

// GET /v1/match-formats
// Lists the global, code-defined match formats. No tenant involved (global data).
func (h *FormatsHandler) ListMatchFormats(w http.ResponseWriter, r *http.Request) {
	formats, err := h.formatService.ListFormats(r.Context())
	if err != nil {
		respondError(r.Context(), w, http.StatusInternalServerError, "Failed to list match formats", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(formats, toMatchFormatDTO))
}

func toMatchFormatDTO(f golf.MatchFormat) sdk.MatchFormat {
	return sdk.MatchFormat{ID: f.ID, Name: f.Name}
}
