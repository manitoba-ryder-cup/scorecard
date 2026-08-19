package rest

import (
	"net/http"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// GET /v1/players
func (r *Router) listPlayers(w http.ResponseWriter, req *http.Request) {
	players, err := r.PlayerService.ListPlayers(req.Context())
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list players", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(players, toPlayerProfileDTO))
}

// POST /v1/players
func (r *Router) createPlayer(w http.ResponseWriter, req *http.Request) {
	// The SDK client validates before sending; this guards non-SDK callers. Domain
	// invariants are enforced separately below.
	body, ok := decodeAndValidate[sdk.CreatePlayerRequest](w, req)
	if !ok {
		return
	}
	player, err := r.PlayerService.CreatePlayer(req.Context(), golf.CreatePlayerInput{
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		UserID:    body.UserID,
	})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to create player", err)
		return
	}
	respondJSON(w, http.StatusCreated, toPlayerDTO(*player))
}

// PUT /v1/players/{id}
// Updates a player's own attributes. Omitted fields keep their stored value.
func (r *Router) updatePlayer(w http.ResponseWriter, req *http.Request) {
	playerID, ok := pathUUIDOr400(w, req, "id", "player")
	if !ok {
		return
	}
	body, ok := decodeAndValidate[sdk.UpdatePlayerRequest](w, req)
	if !ok {
		return
	}
	player, err := r.PlayerService.UpdatePlayer(req.Context(), golf.UpdatePlayerInput{
		ID:        playerID,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
		PhotoPath: body.PhotoPath,
	})
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to update player", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerDTO(*player))
}

// GET /v1/players/{id}
func (r *Router) getPlayer(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "player")
	if !ok {
		return
	}
	player, err := r.PlayerService.GetPlayer(req.Context(), id)
	if err != nil {
		// ErrNotFound -> 404; a real DB failure -> 500 (not masked as "not found").
		respondDomainError(req.Context(), w, "Failed to get player", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerProfileDTO(*player))
}

// GET /v1/players/{id}/stats
func (r *Router) getPlayerStats(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "player")
	if !ok {
		return
	}
	stats, err := r.PlayerService.PlayerStats(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to load player stats", err)
		return
	}
	respondJSON(w, http.StatusOK, toPlayerStatsDTO(*stats))
}

// GET /v1/players/{id}/tournaments
func (r *Router) listPlayerTournaments(w http.ResponseWriter, req *http.Request) {
	id, ok := pathUUIDOr400(w, req, "id", "player")
	if !ok {
		return
	}
	history, err := r.PlayerService.ListPlayerTournaments(req.Context(), id)
	if err != nil {
		respondDomainError(req.Context(), w, "Failed to list player tournaments", err)
		return
	}
	respondJSON(w, http.StatusOK, mapSlice(history, toPlayerTournamentHistoryDTO))
}
