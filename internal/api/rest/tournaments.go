package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type TournamentService interface {
	GetTournament(ctx context.Context, tournamentID uuid.UUID) (*golf.Tournament, error)
	ListTournaments(ctx context.Context) ([]golf.Tournament, error)
	CreateTournament(ctx context.Context, in golf.CreateTournamentInput) (*golf.Tournament, error)
	GetOutcome(ctx context.Context, tournamentID uuid.UUID) (golf.TournamentOutcome, error)
	GetTeamsData(ctx context.Context, tournamentID uuid.UUID) ([]golf.TeamData, error)
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
		respondDomainError(r.Context(), w, "Failed to list tournaments", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(tournaments, toTournamentDTO))
}

// POST /v1/tournaments
func (h *TournamentsHandler) CreateTournament(w http.ResponseWriter, r *http.Request) {
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
	timeZone := req.TimeZone
	if timeZone == "" {
		timeZone = sdk.DefaultTimeZone
	}
	tournament, err := h.tournamentService.CreateTournament(r.Context(), golf.CreateTournamentInput{
		Name:      req.Name,
		TimeZone:  timeZone,
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
func (h *TournamentsHandler) GetTournament(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	tournament, err := h.tournamentService.GetTournament(r.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(r.Context(), w, "Failed to get tournament", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentDTO(*tournament))
}

// GET /v1/tournaments/{id}/teams
func (h *TournamentsHandler) GetTournamentTeams(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	teams, err := h.tournamentService.GetTeamsData(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get teams data", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(teams, toTournamentTeamDTO))
}

// GET /v1/tournaments/{id}/winner
func (h *TournamentsHandler) GetTournamentWinner(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := h.tournamentService.GetOutcome(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get tournament winner", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.WinnerResponse{Finished: outcome.Finished, WinnerTeamID: outcome.WinnerTeamID})
}

// GET /v1/tournaments/{id}/status
func (h *TournamentsHandler) GetTournamentStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	outcome, err := h.tournamentService.GetOutcome(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to check tournament status", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.FinishedResponse{Finished: outcome.Finished})
}
