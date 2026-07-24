package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type TeamService interface {
	SetCaptain(ctx context.Context, teamID, captainID uuid.UUID) (*golf.Team, error)
	ClearCaptain(ctx context.Context, teamID uuid.UUID) error
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
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}
	req, ok := decodeAndValidate[sdk.SetTeamCaptainRequest](w, r)
	if !ok {
		return
	}
	// Unknown team -> 404, unknown player -> 400 (FK), both via respondDomainError.
	if _, err := h.teamService.SetCaptain(r.Context(), teamID, req.CaptainID); err != nil {
		respondDomainError(r.Context(), w, "Failed to set team captain", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /v1/teams/{id}/captain
// Unsets the team's captain (used to reassign); 404 if the team doesn't exist.
func (h *TeamsHandler) ClearCaptain(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}
	if err := h.teamService.ClearCaptain(r.Context(), teamID); err != nil {
		respondDomainError(r.Context(), w, "Failed to clear team captain", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
