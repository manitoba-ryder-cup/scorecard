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
	GetPlayerRecord(ctx context.Context, playerID uuid.UUID) (golf.PlayerRecord, error)
	ListPlayerTournaments(ctx context.Context, playerID uuid.UUID) ([]golf.PlayerTournamentHistory, error)
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
		respondDomainError(r.Context(), w, "Failed to list players", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(players, toPlayerDTO))
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

// GET /v1/players/{id}
func (h *PlayersHandler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}
	player, err := h.playerService.GetPlayer(r.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(r.Context(), w, "Failed to get player", err)
		return
	}
	// The detail view carries the derived W/L/T record; the list view does not.
	record, err := h.playerService.GetPlayerRecord(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to get player record", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerProfileDTO(*player, record))
}

// GET /v1/players/{id}/tournaments
func (h *PlayersHandler) ListPlayerTournaments(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		respondError(r.Context(), w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}
	history, err := h.playerService.ListPlayerTournaments(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list player tournaments", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(history, toPlayerTournamentHistoryDTO))
}
