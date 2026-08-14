package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type PlayerService interface {
	GetPlayer(ctx context.Context, playerID uuid.UUID) (*golf.Player, error)
	ListPlayers(ctx context.Context) ([]golf.Player, error)
	ListPlayerTournaments(ctx context.Context, playerID uuid.UUID) ([]golf.PlayerTournamentHistory, error)
	PlayerStats(ctx context.Context, playerID uuid.UUID) (*golf.PlayerStats, error)
	CreatePlayer(ctx context.Context, in golf.CreatePlayerInput) (*golf.Player, error)
	UpdatePlayer(ctx context.Context, in golf.UpdatePlayerInput) (*golf.Player, error)
}

type PlayersHandler struct {
	playerService PlayerService
}

func NewPlayersHandler(playerService PlayerService) *PlayersHandler {
	return &PlayersHandler{playerService: playerService}
}

// GET /v1/players
func (h *PlayersHandler) ListPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := h.playerService.ListPlayers(r.Context())
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list players", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(players, toPlayerProfileDTO))
}

// POST /v1/players
func (h *PlayersHandler) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	// The SDK client validates before sending; this guards non-SDK callers. Domain
	// invariants are enforced separately below.
	req, ok := decodeAndValidate[sdk.CreatePlayerRequest](w, r)
	if !ok {
		return
	}
	player, err := h.playerService.CreatePlayer(r.Context(), golf.CreatePlayerInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		UserID:    req.UserID,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to create player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toPlayerDTO(*player))
}

// PUT /v1/players/{id}
// Updates a player's own attributes. Omitted fields keep their stored value.
func (h *PlayersHandler) UpdatePlayer(w http.ResponseWriter, r *http.Request) {
	playerID, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.UpdatePlayerRequest](w, r)
	if !ok {
		return
	}
	player, err := h.playerService.UpdatePlayer(r.Context(), golf.UpdatePlayerInput{
		ID:        playerID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		PhotoPath: req.PhotoPath,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to update player", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerDTO(*player))
}

// GET /v1/players/{id}
func (h *PlayersHandler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	player, err := h.playerService.GetPlayer(r.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(r.Context(), w, "Failed to get player", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerProfileDTO(*player))
}

// GET /v1/players/{id}/stats
func (h *PlayersHandler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	stats, err := h.playerService.PlayerStats(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to load player stats", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerStatsDTO(*stats))
}

// GET /v1/players/{id}/tournaments
func (h *PlayersHandler) ListPlayerTournaments(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	history, err := h.playerService.ListPlayerTournaments(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list player tournaments", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(history, toPlayerTournamentHistoryDTO))
}
