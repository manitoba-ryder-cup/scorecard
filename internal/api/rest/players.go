package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

type PlayerService interface {
	GetPlayer(ctx context.Context, playerID uuid.UUID) (*golf.Player, error)
	ListPlayers(ctx context.Context) ([]golf.Player, error)
	GetPlayerRecord(ctx context.Context, playerID uuid.UUID) (golf.PlayerRecord, error)
	CreatePlayer(ctx context.Context, in golf.CreatePlayerInput) (*golf.Player, error)
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
		respondError(w, http.StatusInternalServerError, "Failed to list players", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerDTOs(players))
}

// POST /v1/players
func (h *PlayersHandler) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	var req sdk.CreatePlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	// Server-side shape validation guards non-SDK callers (the SDK client also runs
	// this before sending). Domain invariants are enforced separately below.
	if err := req.Validate(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	player, err := h.playerService.CreatePlayer(r.Context(), golf.CreatePlayerInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		UserID:    req.UserID,
	})
	if err != nil {
		respondDomainError(w, "Failed to create player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toPlayerDTO(*player))
}

// GET /v1/players/{id}
func (h *PlayersHandler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}
	player, err := h.playerService.GetPlayer(r.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(w, "Failed to get player", err)
		return
	}
	// The detail view carries the derived W/L/T record; the list view does not.
	record, err := h.playerService.GetPlayerRecord(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get player record", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerProfileDTO(*player, record))
}
