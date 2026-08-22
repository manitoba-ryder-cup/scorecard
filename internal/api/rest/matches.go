package rest

import (
	"net/http"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/tournaments/{id}/matches
func (r *Router) listMatches(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	matches, err := r.MatchService.ListMatches(req.Context(), tournamentID)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list matches", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(matches, toMatchDTO))
}

// POST /v1/tournaments/{id}/matches
func (r *Router) createMatch(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.CreateMatchRequest](w, req)
	if !ok {
		return
	}
	// Validate already confirmed it parses.
	teeTime, _ := time.Parse(time.RFC3339, body.TeeTime)
	match, err := r.MatchService.CreateMatch(req.Context(), golf.CreateMatchInput{
		TournamentID:  tournamentID,
		CourseID:      body.CourseID,
		TeeColorID:    body.TeeColorID,
		MatchFormatID: body.MatchFormatID,
		TeeTime:       teeTime,
		Handicapped:   body.Handicapped,
	})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to create match", err)
		return
	}
	respondJSON(w, http.StatusCreated, toMatchDTO(*match))
}

// PUT /v1/matches/{id}
func (r *Router) updateMatch(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.UpdateMatchRequest](w, req)
	if !ok {
		return
	}
	in := golf.UpdateMatchInput{
		ID:            id,
		CourseID:      body.CourseID,
		TeeColorID:    body.TeeColorID,
		MatchFormatID: body.MatchFormatID,
		Handicapped:   body.Handicapped,
	}
	if body.TeeTime != nil {
		// Validate already confirmed it parses.
		teeTime, _ := time.Parse(time.RFC3339, *body.TeeTime)
		in.TeeTime = &teeTime
	}
	match, err := r.MatchService.UpdateMatch(req.Context(), in)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to update match", err)
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
func (r *Router) listResults(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	results, phase, err := r.MatchService.ListResults(req.Context(), tournamentID)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list results", err)
		return
	}
	cacheByPhase(w, phase)
	respondJSON(w, http.StatusOK, mapSlice(results, toMatchResultDTO))
}

// GET /v1/matches/{id}/holes
func (r *Router) getMatchHoles(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	holes, err := r.MatchService.ListMatchHoles(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list match holes", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(holes, toHoleDTO))
}

// GET /v1/matches/{id}/participants
func (r *Router) listParticipants(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	participants, err := r.MatchService.ListParticipants(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list participants", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(participants, toMatchParticipantDTO))
}

// POST /v1/matches/{id}/participants
func (r *Router) addParticipant(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.AddParticipantRequest](w, req)
	if !ok {
		return
	}
	participant, err := r.MatchService.AddParticipant(req.Context(), id, body.PlayerID, body.TeamID)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to add participant", err)
		return
	}
	respondJSON(w, http.StatusCreated, toMatchParticipantDTO(*participant))
}

// DELETE /v1/matches/{id}/participants/{playerId}
// Removes a player from the match; 404 if they weren't in it.
func (r *Router) removeParticipant(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	playerID, ok := pathUUIDOr400(w, req, "playerId", "player")
	if !ok {
		return
	}
	if err := r.MatchService.RemoveParticipant(req.Context(), id, playerID); err != nil {
		respondDomainError(req.Context(), w, "Failed to remove participant", err)
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
func (r *Router) getMatchScores(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	scores, err := r.MatchService.CalculateMatchScores(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to calculate match scores", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(scores, toHoleStatusDTO))
}

// POST /v1/matches/{id}/scores
// Records a hole's scores as a unit and recomputes the match's materialized result, which
// DELETE /v1/matches/{id}/scores
// Clears every score on a match and its stored result, leaving the lineup in place. 404 if
// the match doesn't exist; a match with nothing to clear is a 204 like any other, so a
// repeated reset is harmless.
func (r *Router) resetMatch(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	if err := r.MatchService.ResetMatch(req.Context(), id); err != nil {
		respondDomainError(req.Context(), w, "Failed to reset match", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// it returns so the client sees the hole's effect on the match without a second read.
func (r *Router) submitScore(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.ScoreSubmission](w, req)
	if !ok {
		return
	}
	entries := mapSlice(body.Scores, func(s sdk.ScoreEntry) golf.ScoreEntry {
		return golf.ScoreEntry{TeamID: s.TeamID, PlayerID: s.PlayerID, Strokes: s.Strokes}
	})
	// Shape is validated above; the domain still enforces its invariants — team not in
	// the match -> 400, scoring past a finished match -> 409 — while a real failure
	// (DB, etc.) -> 500.
	result, err := r.MatchService.SubmitHoleScores(req.Context(), id, body.HoleNumber, entries)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to submit score", err)
		return
	}
	respondJSON(w, http.StatusOK, toMatchStatusDTO(result))
}

// GET /v1/matches/{id}/winner
// GET /v1/matches/{id}/status
// Both report the match's outcome, in the same shape a score write returns — the winner
// and "is it over" are two questions about one state, so they get one answer.
func (r *Router) getMatchStatus(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	status, err := r.MatchService.MatchStatus(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to get match status", err)
		return
	}
	respondJSON(w, http.StatusOK, toMatchStatusDTO(status))
}
