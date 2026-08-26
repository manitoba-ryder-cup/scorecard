package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

func (r *Router) listTournaments(w http.ResponseWriter, req *http.Request) {
	tournaments, err := r.TournamentService.ListTournaments(req.Context())
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(tournaments, toTournamentDTO))
}

func (r *Router) createTournament(w http.ResponseWriter, req *http.Request) {
	body, ok := decodeAndValidate[sdk.CreateTournamentRequest](w, req)
	if !ok {
		return
	}
	// Unreachable while Validate agrees, but a format skew must not yield a zero date.
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
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusCreated, toTournamentDTO(*tournament))
}

func (r *Router) getTournament(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	tournament, err := r.TournamentService.GetTournament(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	// The record carries the phase, so it goes stale with the leaderboard beside it.
	cacheByPhase(w, tournament.Phase)
	respondJSON(w, http.StatusOK, toTournamentDTO(*tournament))
}

func (r *Router) getTournamentTeams(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	teams, phase, err := r.TournamentService.GetTeamsData(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	cacheByPhase(w, phase)
	respondJSON(w, http.StatusOK, mapSlice(teams, toTournamentTeamDTO))
}

func (r *Router) getTournamentWinner(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := r.TournamentService.GetOutcome(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.WinnerResponse{Finished: outcome.Finished, WinnerTeamID: outcome.WinnerTeamID})
}

func (r *Router) getTournamentStatus(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := r.TournamentService.GetOutcome(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.FinishedResponse{Finished: outcome.Finished})
}
