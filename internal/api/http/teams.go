package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type TeamService interface {
	CreateTeam(ctx context.Context, in golf.CreateTeamInput) (*golf.Team, error)
}

type TeamsHandler struct {
	teamService TeamService
}

func NewTeamsHandler(teamService TeamService) *TeamsHandler {
	return &TeamsHandler{teamService: teamService}
}

// POST /v1/tournaments/{id}/teams
// Creates one of the tournament's two sides. The tournament comes from the path.
func (h *TeamsHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	var req sdk.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	team, err := h.teamService.CreateTeam(r.Context(), golf.CreateTeamInput{
		TournamentID: tournamentID,
		Color:        req.Color,
		CaptainID:    req.CaptainID,
	})
	if err != nil {
		respondDomainError(w, "Failed to create team", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeamDTO(*team))
}
