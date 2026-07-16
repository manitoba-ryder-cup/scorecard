package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type MatchService interface {
	GetWinner(ctx context.Context, matchID int32) (string, error)
	IsFinished(ctx context.Context, matchID int32) (bool, error)
	CalculateMatchScores(ctx context.Context, matchID int32) ([]golf.HoleResult, error)
}

type MatchesHandler struct {
	matchService MatchService
}

func NewMatchesHandler(matchService MatchService) *MatchesHandler {
	return &MatchesHandler{matchService: matchService}
}

// GET /v1/matches/{id}/scores
// Returns hole-by-hole match progression
func (h *MatchesHandler) GetMatchScores(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}

	scores, err := h.matchService.CalculateMatchScores(r.Context(), int32(id))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to calculate match scores", err)
		return
	}

	respondJSON(w, http.StatusOK, scores)
}

// GET /v1/matches/{id}/winner
// Returns the winning team name
func (h *MatchesHandler) GetMatchWinner(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}

	winner, err := h.matchService.GetWinner(r.Context(), int32(id))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to get match winner", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"winner": winner})
}

// GET /v1/matches/{id}/status
// Returns whether the match is finished
func (h *MatchesHandler) GetMatchStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}

	finished, err := h.matchService.IsFinished(r.Context(), int32(id))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check match status", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"finished": finished})
}
