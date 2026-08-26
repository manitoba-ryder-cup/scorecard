package rest

import (
	"net/http"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

func (r *Router) listMatches(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	matches, err := r.MatchService.ListMatches(req.Context(), tournamentID)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(matches, toMatchDTO))
}

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
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusCreated, toMatchDTO(*match))
}

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
		ID:          id,
		CourseID:    body.CourseID,
		TeeColorID:  body.TeeColorID,
		Handicapped: body.Handicapped,
	}
	if body.TeeTime != nil {
		// Validate already confirmed it parses.
		teeTime, _ := time.Parse(time.RFC3339, *body.TeeTime)
		in.TeeTime = &teeTime
	}
	match, err := r.MatchService.UpdateMatch(req.Context(), in)
	if err != nil {
		respondDomainError(req.Context(), w, err)
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

func (r *Router) listResults(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	results, phase, err := r.MatchService.ListResults(req.Context(), tournamentID)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	cacheByPhase(w, phase)
	respondJSON(w, http.StatusOK, mapSlice(results, toMatchResultDTO))
}

func (r *Router) getMatchHoles(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	holes, err := r.MatchService.ListMatchHoles(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(holes, toHoleDTO))
}

func (r *Router) listParticipants(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	participants, err := r.MatchService.ListParticipants(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(participants, toMatchParticipantDTO))
}

func (r *Router) setLineup(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.SetLineupRequest](w, req)
	if !ok {
		return
	}
	entries := make([]golf.MatchParticipant, len(body.Participants))
	for i, e := range body.Participants {
		entries[i] = golf.MatchParticipant{MatchID: id, PlayerID: e.PlayerID, TeamID: e.TeamID}
	}
	if err := r.MatchService.SetLineup(req.Context(), id, entries); err != nil {
		respondDomainError(req.Context(), w, err)
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

// getMatchScores returns the hole-by-hole match-play state.
func (r *Router) getMatchScores(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	scores, err := r.MatchService.CalculateMatchScores(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(scores, toHoleStatusDTO))
}

// submitScore records a hole's scores as a unit and recomputes the match's materialized
// result, which it returns so the client sees the hole's effect without a second read.
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
	// Shape is validated above; the domain still owns its own invariants.
	result, err := r.MatchService.SubmitHoleScores(req.Context(), id, body.HoleNumber, entries)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, toMatchStatusDTO(result))
}

// deleteMatch removes a match and its lineup. A scored match is refused rather than
// taking its results with it; reset it first.
func (r *Router) deleteMatch(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	if err := r.MatchService.DeleteMatch(req.Context(), id); err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resetMatch clears a match's scores. Nothing to clear is a 204, not a 404, so a repeated
// reset is harmless.
func (r *Router) resetMatch(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	if err := r.MatchService.ResetMatch(req.Context(), id); err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getMatchStatus answers both the status and winner routes: one outcome shape, because the
// winner and "is it over" are two questions about one state. Neither route can be dropped
// without a client change.
func (r *Router) getMatchStatus(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "match")
	if !ok {
		return
	}
	status, err := r.MatchService.MatchStatus(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, err)
		return
	}
	respondJSON(w, http.StatusOK, toMatchStatusDTO(status))
}
