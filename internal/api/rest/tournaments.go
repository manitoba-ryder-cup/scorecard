package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/tournaments
func (r *Router) listTournaments(w http.ResponseWriter, req *http.Request) {
	tournaments, err := r.TournamentService.ListTournaments(req.Context())
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list tournaments", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(tournaments, toTournamentDTO))
}

// POST /v1/tournaments
func (r *Router) createTournament(w http.ResponseWriter, req *http.Request) {
	body, ok := decodeAndValidate[sdk.CreateTournamentRequest](w, req)
	if !ok {
		return
	}
	// Validate already confirmed the dates parse; the errors here are unreachable in
	// practice but kept so a future format skew can't silently produce zero dates.
	start, err := parseDate(body.StartDate)
	if err != nil {
		respondError(req.Context(), w, http.StatusBadRequest, "Invalid start_date (want YYYY-MM-DD)", err)
		return
	}
	end, err := parseDate(body.EndDate)
	if err != nil {
		respondError(req.Context(), w, http.StatusBadRequest, "Invalid end_date (want YYYY-MM-DD)", err)
		return
	}
	tournament, err := r.TournamentService.CreateTournament(req.Context(), golf.CreateTournamentInput{
		Name:      body.Name,
		StartDate: start,
		EndDate:   end,
		Location:  body.Location,
	})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to create tournament", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTournamentDTO(*tournament))
}

// GET /v1/tournaments/{id}
func (r *Router) getTournament(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	tournament, err := r.TournamentService.GetTournament(req.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(req.Context(), w, "Failed to get tournament", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentDTO(*tournament))
}

// GET /v1/tournaments/{id}/teams
func (r *Router) getTournamentTeams(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	teams, finished, err := r.TournamentService.GetTeamsData(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to get teams data", err)
		return
	}
	// A finished cup is settled for good; a live one is polled every twenty seconds.
	if finished {
		cacheSettled(w)
	} else {
		cacheLive(w)
	}
	respondJSON(w, http.StatusOK, mapSlice(teams, toTournamentTeamDTO))
}

// GET /v1/tournaments/{id}/winner
func (r *Router) getTournamentWinner(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := r.TournamentService.GetOutcome(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to get tournament winner", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.WinnerResponse{Finished: outcome.Finished, WinnerTeamID: outcome.WinnerTeamID})
}

// GET /v1/tournaments/{id}/status
func (r *Router) getTournamentStatus(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := r.TournamentService.GetOutcome(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to check tournament status", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.FinishedResponse{Finished: outcome.Finished})
}
