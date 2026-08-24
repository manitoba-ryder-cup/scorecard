package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// PUT /v1/teams/{id}/captain
func (r *Router) setCaptain(w http.ResponseWriter, req *http.Request) {
	teamID, ok := pathUUIDOr400(w, req, "id", "team")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.SetTeamCaptainRequest](w, req)
	if !ok {
		return
	}
	if _, err := r.TeamService.SetCaptain(req.Context(), teamID, body.CaptainID); err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /v1/teams/{id}/captain
// Unsets the team's captain (used to reassign); 404 if the team doesn't exist.
func (r *Router) clearCaptain(w http.ResponseWriter, req *http.Request) {
	teamID, ok := pathUUIDOr400(w, req, "id", "team")
	if !ok {
		return
	}
	if err := r.TeamService.ClearCaptain(req.Context(), teamID); err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
