package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// This file maps domain types to SDK DTOs. Handlers must never serialize domain
// structs directly — the wire format is the SDK's deliberate contract, and internal
// fields (e.g. tenant_id) are dropped here.

func toPlayerDTO(p golf.Player) sdk.Player {
	return sdk.Player{
		ID:        p.ID,
		UserID:    p.UserID,
		Email:     p.Email,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		PhotoPath: p.PhotoPath,
	}
}

// toPlayerProfileDTO combines a player with their derived record for the detail view.
func toPlayerProfileDTO(p golf.Player, rec golf.PlayerRecord) sdk.PlayerProfile {
	return sdk.PlayerProfile{
		Player: toPlayerDTO(p),
		Record: sdk.PlayerRecord{
			Wins:   rec.Wins,
			Losses: rec.Losses,
			Ties:   rec.Ties,
		},
	}
}

func toPlayerDTOs(players []golf.Player) []sdk.Player {
	out := make([]sdk.Player, len(players))
	for i, p := range players {
		out[i] = toPlayerDTO(p)
	}
	return out
}

func toTournamentDTO(t golf.Tournament) sdk.Tournament {
	return sdk.Tournament{
		ID:        t.ID,
		Name:      t.Name,
		StartDate: dateString(t.StartDate),
		EndDate:   dateString(t.EndDate),
		Location:  t.Location,
	}
}

func toTournamentDTOs(tournaments []golf.Tournament) []sdk.Tournament {
	out := make([]sdk.Tournament, len(tournaments))
	for i, t := range tournaments {
		out[i] = toTournamentDTO(t)
	}
	return out
}

func toTournamentTeamDTO(td golf.TeamData) sdk.TournamentTeam {
	var captain *sdk.PlayerSummary
	if td.Captain != nil {
		captain = &sdk.PlayerSummary{
			ID:        td.Captain.ID,
			FirstName: td.Captain.FirstName,
			LastName:  td.Captain.LastName,
			Email:     td.Captain.Email,
		}
	}
	return sdk.TournamentTeam{
		ID:      td.ID,
		Color:   td.Color,
		Captain: captain,
		Points:  td.Points,
	}
}

func toTournamentTeamDTOs(teams []golf.TeamData) []sdk.TournamentTeam {
	out := make([]sdk.TournamentTeam, len(teams))
	for i, td := range teams {
		out[i] = toTournamentTeamDTO(td)
	}
	return out
}

func toHoleStatusDTO(h golf.HoleResult) sdk.HoleStatus {
	scores := make([]sdk.TeamHoleScore, len(h.TeamScores))
	for i, ts := range h.TeamScores {
		scores[i] = sdk.TeamHoleScore{TeamID: ts.TeamID, Strokes: ts.Strokes}
	}
	return sdk.HoleStatus{
		HoleNumber:     h.HoleNumber,
		TeamScores:     scores,
		LeaderTeamID:   h.LeaderTeamID,
		Lead:           h.Lead,
		HolesRemaining: h.HolesRemaining,
		Decided:        h.Decided,
	}
}

func toHoleStatusDTOs(holes []golf.HoleResult) []sdk.HoleStatus {
	out := make([]sdk.HoleStatus, len(holes))
	for i, h := range holes {
		out[i] = toHoleStatusDTO(h)
	}
	return out
}

// dateString formats a date as YYYY-MM-DD, or "" if unset.
func dateString(d time.Time) string {
	if d.IsZero() {
		return ""
	}
	return d.Format("2006-01-02")
}

// pathInt parses an int32 path parameter.
func pathInt(r *http.Request, name string) (int32, error) {
	v, err := strconv.ParseInt(r.PathValue(name), 10, 32)
	return int32(v), err
}
