package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/tournaments
func (rt *Router) ListTournaments(w http.ResponseWriter, r *http.Request) {
	tournaments, err := rt.TournamentService.ListTournaments(r.Context())
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list tournaments", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(tournaments, toTournamentDTO))
}

// POST /v1/tournaments
func (rt *Router) CreateTournament(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeAndValidate[sdk.CreateTournamentRequest](w, r)
	if !ok {
		return
	}
	// Validate already confirmed the dates parse; the errors here are unreachable in
	// practice but kept so a future format skew can't silently produce zero dates.
	start, err := parseDate(req.StartDate)
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid start_date (want YYYY-MM-DD)", err)
		return
	}
	end, err := parseDate(req.EndDate)
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid end_date (want YYYY-MM-DD)", err)
		return
	}
	tournament, err := rt.TournamentService.CreateTournament(r.Context(), golf.CreateTournamentInput{
		Name:      req.Name,
		StartDate: start,
		EndDate:   end,
		Location:  req.Location,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to create tournament", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTournamentDTO(*tournament))
}

// GET /v1/tournaments/{id}
func (rt *Router) GetTournament(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	tournament, err := rt.TournamentService.GetTournament(r.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(r.Context(), w, "Failed to get tournament", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentDTO(*tournament))
}

// GET /v1/tournaments/{id}/teams
func (rt *Router) GetTournamentTeams(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	teams, finished, err := rt.TournamentService.GetTeamsData(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get teams data", err)
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
func (rt *Router) GetTournamentWinner(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := rt.TournamentService.GetOutcome(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get tournament winner", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.WinnerResponse{Finished: outcome.Finished, WinnerTeamID: outcome.WinnerTeamID})
}

// GET /v1/tournaments/{id}/status
func (rt *Router) GetTournamentStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := rt.TournamentService.GetOutcome(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to check tournament status", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.FinishedResponse{Finished: outcome.Finished})
}
