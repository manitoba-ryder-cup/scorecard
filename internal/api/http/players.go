package http

import (
	"context"
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

type PlayerService interface {
	GetPlayer(ctx context.Context, playerID int32) (*golf.Player, error)
	ListPlayers(ctx context.Context) ([]golf.Player, error)
	GetPlayerRecord(ctx context.Context, playerID int32) (golf.PlayerRecord, error)
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

// GET /v1/players/{id}
func (h *PlayersHandler) GetPlayer(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid player ID", err)
		return
	}
	player, err := h.playerService.GetPlayer(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Player not found", err)
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
