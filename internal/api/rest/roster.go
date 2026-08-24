package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/tournaments/{id}/players
func (r *Router) listTournamentPlayers(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	players, err := r.RosterService.ListPlayers(req.Context(), tournamentID)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list tournament players", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(players, toTournamentPlayerDTO))
}

// POST /v1/tournaments/{id}/players
func (r *Router) enterPlayer(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.EnterTournamentPlayerRequest](w, req)
	if !ok {
		return
	}
	entry, err := r.RosterService.EnterPlayer(req.Context(), golf.EnterPlayerInput{
		TournamentID: tournamentID,
		PlayerID:     body.PlayerID,
		Tier:         body.Tier,
		Biography:    body.Biography,
		Hdcp:         body.Hdcp,
	})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to enter tournament player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTournamentPlayerDTO(*entry))
}

// PUT /v1/tournaments/{id}/players/{playerId}
func (r *Router) updateTournamentPlayer(w http.ResponseWriter, req *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, req, "id", "tournament")
	if !ok {
		return
	}
	playerID, ok := pathUUIDOr400(w, req, "playerId", "player")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.UpdateTournamentPlayerRequest](w, req)
	if !ok {
		return
	}
	entry, err := r.RosterService.UpdatePlayer(req.Context(), golf.UpdateRosterEntryInput{
		TournamentID: tournamentID,
		PlayerID:     playerID,
		Tier:         body.Tier,
		Biography:    body.Biography,
		Hdcp:         body.Hdcp,
	})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to update tournament player", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentPlayerDTO(*entry))
}

// POST /v1/teams/{id}/members
// Drafts an entered player onto the team (the tournament is the team's).
func (r *Router) draftPlayer(w http.ResponseWriter, req *http.Request) {
	teamID, ok := pathUUIDOr400(w, req, "id", "team")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.DraftPlayerRequest](w, req)
	if !ok {
		return
	}
	member, err := r.RosterService.DraftPlayer(req.Context(), teamID, body.PlayerID)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to draft player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeamMemberDTO(*member))
}

// DELETE /v1/teams/{id}/members/{playerId}
// Undrafts a player from the team; 404 if they weren't on it.
func (r *Router) undraftPlayer(w http.ResponseWriter, req *http.Request) {
	teamID, ok := pathUUIDOr400(w, req, "id", "team")
	if !ok {
		return
	}
	playerID, ok := pathUUIDOr400(w, req, "playerId", "player")
	if !ok {
		return
	}
	if err := r.RosterService.UndraftPlayer(req.Context(), teamID, playerID); err != nil {
		if refusedForScores(req.Context(), w, "That player has been scored in a match. Reset it before undrafting them.", err) {
			return
		}
		respondDomainError(req.Context(), w, "Failed to undraft player", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/teams/{id}/members
func (r *Router) listTeamMembers(w http.ResponseWriter, req *http.Request) {
	teamID, ok := pathUUIDOr400(w, req, "id", "team")
	if !ok {
		return
	}
	members, err := r.RosterService.ListTeamMembers(req.Context(), teamID)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list team members", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(members, toTournamentPlayerDTO))
}

func toTournamentPlayerDTO(tp golf.TournamentPlayer) sdk.TournamentPlayer {
	return sdk.TournamentPlayer{
		TournamentID: tp.TournamentID,
		PlayerID:     tp.PlayerID,
		Tier:         tp.Tier,
		Biography:    tp.Biography,
		Hdcp:         tp.Hdcp,
		FirstName:    tp.FirstName,
		LastName:     tp.LastName,
		PhotoPath:    tp.PhotoPath,
		TeamID:       tp.TeamID,
		Record:       toPlayerRecordDTO(tp.Record),
		CupsWon:      tp.CupsWon,
	}
}

func toTeamMemberDTO(m golf.TeamMember) sdk.TeamMember {
	return sdk.TeamMember{
		TeamID:       m.TeamID,
		PlayerID:     m.PlayerID,
		TournamentID: m.TournamentID,
	}
}
