package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type RosterService interface {
	EnterPlayer(ctx context.Context, in golf.EnterPlayerInput) (*golf.TournamentPlayer, error)
	UpdatePlayer(ctx context.Context, in golf.EnterPlayerInput) (*golf.TournamentPlayer, error)
	ListPlayers(ctx context.Context, tournamentID uuid.UUID) ([]golf.TournamentPlayer, error)
	DraftPlayer(ctx context.Context, teamID, playerID uuid.UUID) (*golf.TeamMember, error)
	UndraftPlayer(ctx context.Context, teamID, playerID uuid.UUID) error
	ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]golf.TournamentPlayer, error)
}

type RosterHandler struct {
	rosterService RosterService
}

func NewRosterHandler(rosterService RosterService) *RosterHandler {
	return &RosterHandler{rosterService: rosterService}
}

func tierOrDefault(tier string) string {
	if tier == "" {
		return sdk.DefaultTier
	}
	return tier
}

// GET /v1/tournaments/{id}/players
func (h *RosterHandler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	players, err := h.rosterService.ListPlayers(r.Context(), tournamentID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list tournament players", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(players, toTournamentPlayerDTO))
}

// POST /v1/tournaments/{id}/players
func (h *RosterHandler) EnterPlayer(w http.ResponseWriter, r *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.EnterTournamentPlayerRequest](w, r)
	if !ok {
		return
	}
	entry, err := h.rosterService.EnterPlayer(r.Context(), golf.EnterPlayerInput{
		TournamentID: tournamentID,
		PlayerID:     req.PlayerID,
		Tier:         tierOrDefault(req.Tier),
		Biography:    req.Biography,
		Hdcp:         req.Hdcp,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to enter tournament player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTournamentPlayerDTO(*entry))
}

// PUT /v1/tournaments/{id}/players/{playerId}
func (h *RosterHandler) UpdatePlayer(w http.ResponseWriter, r *http.Request) {
	tournamentID, ok := pathUUIDOr400(w, r, "id", "tournament")
	if !ok {
		return
	}
	playerID, ok := pathUUIDOr400(w, r, "playerId", "player")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.UpdateTournamentPlayerRequest](w, r)
	if !ok {
		return
	}
	entry, err := h.rosterService.UpdatePlayer(r.Context(), golf.EnterPlayerInput{
		TournamentID: tournamentID,
		PlayerID:     playerID,
		Tier:         tierOrDefault(req.Tier),
		Biography:    req.Biography,
		Hdcp:         req.Hdcp,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to update tournament player", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentPlayerDTO(*entry))
}

// POST /v1/teams/{id}/members
// Drafts an entered player onto the team (the tournament is the team's).
func (h *RosterHandler) DraftPlayer(w http.ResponseWriter, r *http.Request) {
	teamID, ok := pathUUIDOr400(w, r, "id", "team")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.DraftPlayerRequest](w, r)
	if !ok {
		return
	}
	member, err := h.rosterService.DraftPlayer(r.Context(), teamID, req.PlayerID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to draft player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeamMemberDTO(*member))
}

// DELETE /v1/teams/{id}/members/{playerId}
// Undrafts a player from the team; 404 if they weren't on it.
func (h *RosterHandler) UndraftPlayer(w http.ResponseWriter, r *http.Request) {
	teamID, ok := pathUUIDOr400(w, r, "id", "team")
	if !ok {
		return
	}
	playerID, ok := pathUUIDOr400(w, r, "playerId", "player")
	if !ok {
		return
	}
	if err := h.rosterService.UndraftPlayer(r.Context(), teamID, playerID); err != nil {
		respondDomainError(r.Context(), w, "Failed to undraft player", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/teams/{id}/members
func (h *RosterHandler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	teamID, ok := pathUUIDOr400(w, r, "id", "team")
	if !ok {
		return
	}
	members, err := h.rosterService.ListTeamMembers(r.Context(), teamID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list team members", err)
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
