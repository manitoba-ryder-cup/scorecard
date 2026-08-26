package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

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

// clearCaptain leaves the team with no captain; 404 if the team doesn't exist. Naming a
// different one is a PUT, not a delete first.
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
