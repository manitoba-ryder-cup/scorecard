package rest

import (
	"context"
	"encoding/json"
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
	ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]golf.TournamentPlayer, error)
}

type RosterHandler struct {
	rosterService RosterService
}

func NewRosterHandler(rosterService RosterService) *RosterHandler {
	return &RosterHandler{rosterService: rosterService}
}

// defaultTier applies the schema's default when a tier isn't supplied.
func defaultTier(tier string) string {
	if tier == "" {
		return "white"
	}
	return tier
}

// GET /v1/tournaments/{id}/players
func (h *RosterHandler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	players, err := h.rosterService.ListPlayers(r.Context(), tournamentID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list tournament players", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentPlayerDTOs(players))
}

// POST /v1/tournaments/{id}/players
func (h *RosterHandler) EnterPlayer(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	var req sdk.EnterTournamentPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	entry, err := h.rosterService.EnterPlayer(r.Context(), golf.EnterPlayerInput{
		TournamentID: tournamentID,
		PlayerID:     req.PlayerID,
		Tier:         defaultTier(req.Tier),
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
	tournamentID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	playerID, err := pathUUID(r, "playerId")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}
	var req sdk.UpdateTournamentPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	entry, err := h.rosterService.UpdatePlayer(r.Context(), golf.EnterPlayerInput{
		TournamentID: tournamentID,
		PlayerID:     playerID,
		Tier:         defaultTier(req.Tier),
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
	teamID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}
	var req sdk.DraftPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	member, err := h.rosterService.DraftPlayer(r.Context(), teamID, req.PlayerID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to draft player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTeamMemberDTO(*member))
}

// GET /v1/teams/{id}/members
func (h *RosterHandler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid team ID", err)
		return
	}
	members, err := h.rosterService.ListTeamMembers(r.Context(), teamID)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list team members", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentPlayerDTOs(members))
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
		Email:        tp.Email,
		PhotoPath:    tp.PhotoPath,
		TeamID:       tp.TeamID,
	}
}

func toTournamentPlayerDTOs(entries []golf.TournamentPlayer) []sdk.TournamentPlayer {
	out := make([]sdk.TournamentPlayer, len(entries))
	for i, e := range entries {
		out[i] = toTournamentPlayerDTO(e)
	}
	return out
}

func toTeamMemberDTO(m golf.TeamMember) sdk.TeamMember {
	return sdk.TeamMember{
		TeamID:       m.TeamID,
		PlayerID:     m.PlayerID,
		TournamentID: m.TournamentID,
	}
}
