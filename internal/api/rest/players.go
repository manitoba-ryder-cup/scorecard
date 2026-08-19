package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/players
func (rt *Router) ListPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := rt.PlayerService.ListPlayers(r.Context())
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list players", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(players, toPlayerProfileDTO))
}

// POST /v1/players
func (rt *Router) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	// The SDK client validates before sending; this guards non-SDK callers. Domain
	// invariants are enforced separately below.
	req, ok := decodeAndValidate[sdk.CreatePlayerRequest](w, r)
	if !ok {
		return
	}
	player, err := rt.PlayerService.CreatePlayer(r.Context(), golf.CreatePlayerInput{
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
func (rt *Router) UpdatePlayer(w http.ResponseWriter, r *http.Request) {
	playerID, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[sdk.UpdatePlayerRequest](w, r)
	if !ok {
		return
	}
	player, err := rt.PlayerService.UpdatePlayer(r.Context(), golf.UpdatePlayerInput{
		ID:        playerID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		PhotoPath: req.PhotoPath,
	})
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to update player", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerDTO(*player))
}

// GET /v1/players/{id}
func (rt *Router) GetPlayer(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	player, err := rt.PlayerService.GetPlayer(r.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(r.Context(), w, "Failed to get player", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerProfileDTO(*player))
}

// GET /v1/players/{id}/stats
func (rt *Router) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	stats, err := rt.PlayerService.PlayerStats(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to load player stats", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerStatsDTO(*stats))
}

// GET /v1/players/{id}/tournaments
func (rt *Router) ListPlayerTournaments(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUIDOr400(w, r, "id", "player")
	if !ok {
		return
	}
	history, err := rt.PlayerService.ListPlayerTournaments(r.Context(), id)
	if err != nil {
		respondDomainError(r.Context(), w, "Failed to list player tournaments", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(history, toPlayerTournamentHistoryDTO))
}
