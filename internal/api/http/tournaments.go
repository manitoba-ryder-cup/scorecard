package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type TournamentService interface {
	GetTournament(ctx context.Context, tournamentID int32) (*golf.Tournament, error)
	ListTournaments(ctx context.Context) ([]golf.Tournament, error)
	IsFinished(ctx context.Context, tournamentID int32) (bool, error)
	GetWinningTeam(ctx context.Context, tournamentID int32) (string, error)
	GetTeamsData(ctx context.Context, tournamentID int32) ([]golf.TeamData, error)
}

type TournamentsHandler struct {
	tournamentService TournamentService
}

func NewTournamentsHandler(tournamentService TournamentService) *TournamentsHandler {
	return &TournamentsHandler{tournamentService: tournamentService}
}

// GET /v1/tournaments
func (h *TournamentsHandler) ListTournaments(w http.ResponseWriter, r *http.Request) {
	tournaments, err := h.tournamentService.ListTournaments(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list tournaments", err)
		return
	}

	respondJSON(w, http.StatusOK, tournaments)
}

// GET /v1/tournaments/{id}
func (h *TournamentsHandler) GetTournament(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}

	tournament, err := h.tournamentService.GetTournament(r.Context(), int32(id))
	if err != nil {
		respondError(w, http.StatusNotFound, "Tournament not found", err)
		return
	}

	respondJSON(w, http.StatusOK, tournament)
}

// GET /v1/tournaments/{id}/teams
// Returns teams with captain and points
func (h *TournamentsHandler) GetTournamentTeams(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}

	teams, err := h.tournamentService.GetTeamsData(r.Context(), int32(id))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get teams data", err)
		return
	}

	respondJSON(w, http.StatusOK, teams)
}

// GET /v1/tournaments/{id}/winner
// Returns the winning team name
func (h *TournamentsHandler) GetTournamentWinner(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}

	winner, err := h.tournamentService.GetWinningTeam(r.Context(), int32(id))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to get tournament winner", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"winner": winner})
}

// GET /v1/tournaments/{id}/status
// Returns whether the tournament is finished
func (h *TournamentsHandler) GetTournamentStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}

	finished, err := h.tournamentService.IsFinished(r.Context(), int32(id))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check tournament status", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"finished": finished})
}
