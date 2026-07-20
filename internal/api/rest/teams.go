package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type TeamService interface {
	SetCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*golf.Team, error)
}

type TeamsHandler struct {
	teamService TeamService
}

func NewTeamsHandler(teamService TeamService) *TeamsHandler {
	return &TeamsHandler{teamService: teamService}
}

// PUT /v1/teams/{id}/captain
func (h *TeamsHandler) SetCaptain(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}
	var req sdk.SetTeamCaptainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	// Unknown team -> 404, unknown player -> 400 (FK), both via respondDomainError.
	if _, err := h.teamService.SetCaptain(r.Context(), teamID, req.CaptainID); err != nil {
		respondDomainError(w, "Failed to set team captain", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
