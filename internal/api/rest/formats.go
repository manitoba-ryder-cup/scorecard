package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// listMatchFormats lists the seeded match formats. No tenant involved (global data).
func (r *Router) listMatchFormats(w http.ResponseWriter, req *http.Request) {
	formats, err := r.FormatService.ListFormats(req.Context())
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(formats, toMatchFormatDTO))
}

func toMatchFormatDTO(f golf.MatchFormat) sdk.MatchFormat {
	return sdk.MatchFormat{ID: f.ID, Name: f.Name, PlayersPerSide: f.PlayersPerSide, ScoresPerPlayer: f.ScoresPerPlayer}
}
