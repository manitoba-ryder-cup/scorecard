package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type RosterService interface {
	EnterPlayer(ctx context.Context, in golf.EnterPlayerInput) (*golf.TournamentPlayer, error)
	UpdatePlayer(ctx context.Context, in golf.EnterPlayerInput) (*golf.TournamentPlayer, error)
	ListPlayers(ctx context.Context, tournamentID int32) ([]golf.TournamentPlayerDetail, error)
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
	tournamentID, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	players, err := h.rosterService.ListPlayers(r.Context(), tournamentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list tournament players", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentPlayerDetailDTOs(players))
}

// POST /v1/tournaments/{id}/players
func (h *RosterHandler) EnterPlayer(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	var req sdk.EnterTournamentPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
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
		respondDomainError(w, "Failed to enter tournament player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toTournamentPlayerDTO(*entry))
}

// PUT /v1/tournaments/{id}/players/{playerId}
func (h *RosterHandler) UpdatePlayer(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tournament ID", err)
		return
	}
	playerID, err := pathInt(r, "playerId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}
	var req sdk.UpdateTournamentPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
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
		respondDomainError(w, "Failed to update tournament player", err)
		return
	}
	respondJSON(w, http.StatusOK, toTournamentPlayerDTO(*entry))
}

func toTournamentPlayerDTO(tp golf.TournamentPlayer) sdk.TournamentPlayer {
	return sdk.TournamentPlayer{
		TournamentID: tp.TournamentID,
		PlayerID:     tp.PlayerID,
		Tier:         tp.Tier,
		Biography:    tp.Biography,
		Hdcp:         tp.Hdcp,
	}
}

func toTournamentPlayerDetailDTOs(entries []golf.TournamentPlayerDetail) []sdk.TournamentPlayerDetail {
	out := make([]sdk.TournamentPlayerDetail, len(entries))
	for i, e := range entries {
		out[i] = sdk.TournamentPlayerDetail{
			TournamentPlayer: toTournamentPlayerDTO(e.TournamentPlayer),
			FirstName:        e.FirstName,
			LastName:         e.LastName,
			Email:            e.Email,
			PhotoPath:        e.PhotoPath,
		}
	}
	return out
}
