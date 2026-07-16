package http

import (
	"context"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type MatchService interface {
	GetWinner(ctx context.Context, matchID int32) (*int32, error)
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
// Returns the hole-by-hole match-play state.
func (h *MatchesHandler) GetMatchScores(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	scores, err := h.matchService.CalculateMatchScores(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to calculate match scores", err)
		return
	}
	respondJSON(w, http.StatusOK, toHoleStatusDTOs(scores))
}

// GET /v1/matches/{id}/winner
func (h *MatchesHandler) GetMatchWinner(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	finished, err := h.matchService.IsFinished(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check match status", err)
		return
	}
	winnerID, err := h.matchService.GetWinner(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get match winner", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.MatchWinnerResponse{Finished: finished, WinnerTeamID: winnerID})
}

// GET /v1/matches/{id}/status
func (h *MatchesHandler) GetMatchStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	finished, err := h.matchService.IsFinished(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check match status", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.FinishedResponse{Finished: finished})
}
