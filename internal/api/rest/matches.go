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
	MatchStatus(ctx context.Context, matchID uuid.UUID) (golf.StoredResult, error)
	CalculateMatchScores(ctx context.Context, matchID uuid.UUID) ([]golf.HoleResult, error)
	SubmitHoleScores(ctx context.Context, matchID uuid.UUID, hole int32, entries []golf.ScoreEntry) (golf.StoredResult, error)
	CreateMatch(ctx context.Context, in golf.CreateMatchInput) (*golf.Match, error)
	UpdateMatch(ctx context.Context, in golf.UpdateMatchInput) (*golf.Match, error)
	ListMatches(ctx context.Context, tournamentID uuid.UUID) ([]golf.Match, error)
	ListMatchHoles(ctx context.Context, matchID uuid.UUID) ([]golf.Hole, error)
	ListResults(ctx context.Context, tournamentID uuid.UUID) ([]golf.MatchResult, error)
	AddParticipant(ctx context.Context, matchID, playerID, teamID uuid.UUID) (*golf.MatchParticipant, error)
	RemoveParticipant(ctx context.Context, matchID, playerID uuid.UUID) error
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
	tournamentID, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
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
	tournamentID, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.CreateMatchRequest](w, r)
	if !ok {
		return
	}
	// Validate already confirmed it parses.
	teeTime, _ := time.Parse(time.RFC3339, req.TeeTime)
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

// PUT /v1/matches/{id}
func (h *MatchesHandler) UpdateMatch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.UpdateMatchRequest](w, r)
	if !ok {
		return
	}
	in := golf.UpdateMatchInput{
		ID:            id,
		CourseID:      req.CourseID,
		TeeColorID:    req.TeeColorID,
		MatchFormatID: req.MatchFormatID,
		Handicapped:   req.Handicapped,
	}
	if req.TeeTime != nil {
		// Validate already confirmed it parses.
		teeTime, _ := time.Parse(time.RFC3339, *req.TeeTime)
		in.TeeTime = &teeTime
	}
	match, err := h.matchService.UpdateMatch(r.Context(), in)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to update match", err)
		return
	}
	respondJSON(w, http.StatusOK, toMatchDTO(*match))
}

func toMatchDTO(m golf.Match) sdk.Match {
	return sdk.Match{
		ID:            m.ID,
		TournamentID:  m.TournamentID,
		CourseID:      m.CourseID,
		TeeColorID:    m.TeeColorID,
		MatchFormatID: m.MatchFormatID,
		TeeTime:       m.TeeTime.Format(time.RFC3339),
		Handicapped:   m.Handicapped,
	}
}

// GET /v1/tournaments/{id}/results
func (h *MatchesHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	results, err := h.matchService.ListResults(r.Context(), tournamentID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list results", err)
		return
	}
	// Answered from what we already hold: once every match has finished nothing here can
	// change again. Until then a spectator is polling this every twenty seconds, so
	// caching it at all would defeat the poll.
	if allFinished(results) {
		cacheSettled(w)
	} else {
		cacheLive(w)
	}
	respondJSON(w, http.StatusOK, mapSlice(results, toMatchResultDTO))
}

// allFinished reports whether every match in a tournament has a settled result. An empty
// schedule is not finished — a cup with no matches yet is one that hasn't started.
func allFinished(results []golf.MatchResult) bool {
	if len(results) == 0 {
		return false
	}
	for _, m := range results {
		if !m.Finished {
			return false
		}
	}
	return true
}

// GET /v1/matches/{id}/holes
func (h *MatchesHandler) GetMatchHoles(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
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
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
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
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
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

// DELETE /v1/matches/{id}/participants/{playerId}
// Removes a player from the match; 404 if they weren't in it.
func (h *MatchesHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
		return
	}
	playerID, ok := pathUUIDOr400(w, r, "playerId", "player")
	if !ok {
		return
	}
	if err := h.matchService.RemoveParticipant(r.Context(), id, playerID); err != nil {
		respondDomainError(r.Context(), w, "Failed to remove participant", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
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
// Records a hole's scores as a unit and recomputes the match's materialized result, which
// it returns so the client sees the hole's effect on the match without a second read.
func (h *MatchesHandler) SubmitScore(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.ScoreSubmission](w, r)
	if !ok {
		return
	}
	entries := mapSlice(req.Scores, func(s sdk.ScoreEntry) golf.ScoreEntry {
		return golf.ScoreEntry{TeamID: s.TeamID, PlayerID: s.PlayerID, Strokes: s.Strokes}
	})
	// Shape is validated above; the domain still enforces its invariants — team not in
	// the match -> 400, scoring past a finished match -> 409 — while a real failure
	// (DB, etc.) -> 500.
	result, err := h.matchService.SubmitHoleScores(r.Context(), id, req.HoleNumber, entries)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to submit score", err)
		return
	}
	respondJSON(w, http.StatusOK, toMatchStatusDTO(result))
}

// GET /v1/matches/{id}/winner
// GET /v1/matches/{id}/status
// Both report the match's outcome, in the same shape a score write returns — the winner
// and "is it over" are two questions about one state, so they get one answer.
func (h *MatchesHandler) GetMatchStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "match")
	if !ok {
		return
	}
	status, err := h.matchService.MatchStatus(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get match status", err)
		return
	}
	respondJSON(w, http.StatusOK, toMatchStatusDTO(status))
}
