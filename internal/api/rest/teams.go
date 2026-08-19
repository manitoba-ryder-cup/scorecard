package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// PUT /v1/teams/{id}/captain
func (rt *Router) SetCaptain(w http.ResponseWriter, r *http.Request) {
	teamID, ok := pathUUIDOr400(w, r, "id", "team")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.SetTeamCaptainRequest](w, r)
	if !ok {
		return
	}
	// Unknown team -> 404, unknown player -> 400 (FK), both via respondDomainError.
	if _, err := rt.TeamService.SetCaptain(r.Context(), teamID, req.CaptainID); err != nil {
		respondDomainError(r.Context(), w, "Failed to set team captain", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /v1/teams/{id}/captain
// Unsets the team's captain (used to reassign); 404 if the team doesn't exist.
func (rt *Router) ClearCaptain(w http.ResponseWriter, r *http.Request) {
	teamID, ok := pathUUIDOr400(w, r, "id", "team")
	if !ok {
		return
	}
	if err := rt.TeamService.ClearCaptain(r.Context(), teamID); err != nil {
		respondDomainError(r.Context(), w, "Failed to clear team captain", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
