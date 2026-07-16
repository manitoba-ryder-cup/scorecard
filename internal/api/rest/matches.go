package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type MatchService interface {
	GetWinner(ctx context.Context, matchID int32) (*int32, error)
	IsFinished(ctx context.Context, matchID int32) (bool, error)
	CalculateMatchScores(ctx context.Context, matchID int32) ([]golf.HoleResult, error)
	SubmitScore(ctx context.Context, matchID int32, entry golf.ScoreEntry) error
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

// POST /v1/matches/{id}/scores
// Records one hole score and recomputes the match's materialized result.
func (h *MatchesHandler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	var req sdk.ScoreSubmission
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	entry := golf.ScoreEntry{
		HoleNumber: req.HoleNumber,
		Strokes:    req.Strokes,
		TeamID:     req.TeamID,
		PlayerID:   req.PlayerID,
	}
	// Shape is validated above; the domain still enforces its invariant (the team must
	// be in the match) -> 400, while a real failure (DB, etc.) -> 500.
	if err := h.matchService.SubmitScore(r.Context(), id, entry); err != nil {
		respondDomainError(w, "Failed to submit score", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
