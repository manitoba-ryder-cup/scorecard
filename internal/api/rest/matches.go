package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type MatchService interface {
	GetWinner(ctx context.Context, matchID uuid.UUID) (*uuid.UUID, error)
	IsFinished(ctx context.Context, matchID uuid.UUID) (bool, error)
	CalculateMatchScores(ctx context.Context, matchID uuid.UUID) ([]golf.HoleResult, error)
	SubmitScore(ctx context.Context, matchID uuid.UUID, entry golf.ScoreEntry) error
	CreateMatch(ctx context.Context, in golf.CreateMatchInput) (*golf.Match, error)
	ListMatches(ctx context.Context, tournamentID uuid.UUID) ([]golf.Match, error)
	ListMatchHoles(ctx context.Context, matchID uuid.UUID) ([]golf.Hole, error)
	ListResults(ctx context.Context, tournamentID uuid.UUID) ([]golf.MatchResult, error)
	AddParticipant(ctx context.Context, matchID, playerID, teamID uuid.UUID) (*golf.MatchParticipant, error)
	ListParticipants(ctx context.Context, matchID uuid.UUID) ([]golf.MatchParticipant, error)
}

type MatchesHandler struct {
	matchService MatchService
}

func NewMatchesHandler(matchService MatchService) *MatchesHandler {
	return &MatchesHandler{matchService: matchService}
}

// GET /v1/tournaments/{id}/matches
func (h *MatchesHandler) ListMatches(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	matches, err := h.matchService.ListMatches(r.Context(), tournamentID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list matches", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(matches, toMatchDTO))
}

// POST /v1/tournaments/{id}/matches
func (h *MatchesHandler) CreateMatch(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	req, ok := decodeAndValidate[sdk.CreateMatchRequest](w, r)
	if !ok {
		return
	}
	var teeTime *time.Time
	if req.TeeTime != nil {
		// Validate already confirmed it parses.
		tt, _ := time.Parse(time.RFC3339, *req.TeeTime)
		teeTime = &tt
	}
	match, err := h.matchService.CreateMatch(r.Context(), golf.CreateMatchInput{
		TournamentID:  tournamentID,
		CourseID:      req.CourseID,
		TeeColorID:    req.TeeColorID,
		MatchFormatID: req.MatchFormatID,
		TeeTime:       teeTime,
		Handicapped:   req.Handicapped,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to create match", err)
		return
	}
	respondJSON(w, http.StatusCreated, toMatchDTO(*match))
}

func toMatchDTO(m golf.Match) sdk.Match {
	var teeTime *string
	if m.TeeTime != nil {
		s := m.TeeTime.Format(time.RFC3339)
		teeTime = &s
	}
	return sdk.Match{
		ID:            m.ID,
		TournamentID:  m.TournamentID,
		CourseID:      m.CourseID,
		TeeColorID:    m.TeeColorID,
		MatchFormatID: m.MatchFormatID,
		TeeTime:       teeTime,
		Handicapped:   m.Handicapped,
	}
}

// GET /v1/tournaments/{id}/results
func (h *MatchesHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	results, err := h.matchService.ListResults(r.Context(), tournamentID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list results", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(results, toMatchResultDTO))
}

// GET /v1/matches/{id}/holes
func (h *MatchesHandler) GetMatchHoles(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	holes, err := h.matchService.ListMatchHoles(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list match holes", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(holes, toHoleDTO))
}

// GET /v1/matches/{id}/participants
func (h *MatchesHandler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	participants, err := h.matchService.ListParticipants(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list participants", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(participants, toMatchParticipantDTO))
}

// POST /v1/matches/{id}/participants
func (h *MatchesHandler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	req, ok := decodeAndValidate[sdk.AddParticipantRequest](w, r)
	if !ok {
		return
	}
	participant, err := h.matchService.AddParticipant(r.Context(), id, req.PlayerID, req.TeamID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to add participant", err)
		return
	}
	respondJSON(w, http.StatusCreated, toMatchParticipantDTO(*participant))
}

func toMatchParticipantDTO(p golf.MatchParticipant) sdk.MatchParticipant {
	return sdk.MatchParticipant{
		TournamentID: p.TournamentID,
		MatchID:      p.MatchID,
		PlayerID:     p.PlayerID,
		TeamID:       p.TeamID,
	}
}

// GET /v1/matches/{id}/scores
// Returns the hole-by-hole match-play state.
func (h *MatchesHandler) GetMatchScores(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	scores, err := h.matchService.CalculateMatchScores(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to calculate match scores", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(scores, toHoleStatusDTO))
}

// POST /v1/matches/{id}/scores
// Records one hole score and recomputes the match's materialized result.
func (h *MatchesHandler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	req, ok := decodeAndValidate[sdk.ScoreSubmission](w, r)
	if !ok {
		return
	}
	entry := golf.ScoreEntry{
		HoleNumber: req.HoleNumber,
		Strokes:    req.Strokes,
		TeamID:     req.TeamID,
		PlayerID:   req.PlayerID,
	}
	// Shape is validated above; the domain still enforces its invariant (the team must
	// be in the match) -> 400, while a real failure (DB, etc.) -> 500.
	if err := h.matchService.SubmitScore(r.Context(), id, entry); err != nil {
		respondDomainError(r.Context(), w, "Failed to submit score", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/matches/{id}/winner
func (h *MatchesHandler) GetMatchWinner(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	finished, err := h.matchService.IsFinished(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to check match status", err)
		return
	}
	winnerID, err := h.matchService.GetWinner(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get match winner", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.WinnerResponse{Finished: finished, WinnerTeamID: winnerID})
}

// GET /v1/matches/{id}/status
func (h *MatchesHandler) GetMatchStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid match ID", err)
		return
	}
	finished, err := h.matchService.IsFinished(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to check match status", err)
		return
	}
	respondJSON(w, http.StatusOK, sdk.FinishedResponse{Finished: finished})
}
