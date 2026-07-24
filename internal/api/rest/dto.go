package rest

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// This file maps domain types to SDK DTOs. Handlers must never serialize domain
// structs directly — the wire format is the SDK's deliberate contract, and internal
// fields (e.g. tenant_id) are dropped here.

// mapSlice converts a slice by applying f to each element — used to lift a per-item
// DTO mapper over a list without a bespoke loop per type.
func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

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

// toPlayerProfileDTO is the full player wire shape: identity plus the derived record
// and cups won.
func toPlayerProfileDTO(p golf.Player) sdk.PlayerProfile {
	return sdk.PlayerProfile{
		Player:  toPlayerDTO(p),
		Record:  toPlayerRecordDTO(p.Record),
		CupsWon: p.CupsWon,
	}
}

func toPlayerRecordDTO(rec golf.PlayerRecord) sdk.PlayerRecord {
	return sdk.PlayerRecord{Wins: rec.Wins, Losses: rec.Losses, Ties: rec.Ties}
}

func toPlayerTournamentHistoryDTO(h golf.PlayerTournamentHistory) sdk.PlayerTournamentHistory {
	return sdk.PlayerTournamentHistory{
		TournamentID:     h.TournamentID,
		Name:             h.Name,
		Location:         h.Location,
		StartDate:        dateString(h.StartDate),
		EndDate:          dateString(h.EndDate),
		CaptainFirstName: h.CaptainFirstName,
		CaptainLastName:  h.CaptainLastName,
		Result:           h.Result,
		Record:           toPlayerRecordDTO(h.Record),
	}
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

func toHoleDTO(h golf.Hole) sdk.Hole {
	return sdk.Hole{Number: h.Number, Par: h.Par, Hdcp: h.Hdcp, Yards: h.Yards}
}

func toMatchResultDTO(m golf.MatchResult) sdk.MatchResult {
	var teeTime *string
	if m.TeeTime != nil {
		s := m.TeeTime.Format(time.RFC3339)
		teeTime = &s
	}
	// Empty (not null) so the client always gets an array to iterate.
	holeResults := m.HoleResults
	if holeResults == nil {
		holeResults = []*uuid.UUID{}
	}
	return sdk.MatchResult{
		MatchID:        m.MatchID,
		FormatName:     m.FormatName,
		Finished:       m.Finished,
		WinnerTeamID:   m.WinnerTeamID,
		Lead:           m.Lead,
		HolesRemaining: m.HolesRemaining,
		Sides:          mapSlice(m.Sides, toMatchSideDTO),
		HoleResults:    holeResults,
		TeeTime:        teeTime,
		CourseName:     m.CourseName,
	}
}

func toMatchSideDTO(s golf.MatchSide) sdk.MatchSide {
	return sdk.MatchSide{
		TeamID: s.TeamID,
		Players: mapSlice(s.Players, func(p golf.MatchSidePlayer) sdk.MatchPlayer {
			return sdk.MatchPlayer{PlayerID: p.PlayerID, FirstName: p.FirstName, LastName: p.LastName}
		}),
	}
}

func toTeeColorDTO(tc golf.TeeColor) sdk.TeeColor {
	return sdk.TeeColor{ID: tc.ID, Color: tc.Color}
}

func toTeeSetDTO(ts golf.TeeSetWithHoles) sdk.TeeSet {
	holes := make([]sdk.Hole, len(ts.Holes))
	for i, h := range ts.Holes {
		holes[i] = sdk.Hole{Number: h.Number, Par: h.Par, Hdcp: h.Hdcp, Yards: h.Yards}
	}
	return sdk.TeeSet{
		CourseID:   ts.TeeSet.CourseID,
		TeeColorID: ts.TeeSet.TeeColorID,
		Slope:      ts.TeeSet.Slope,
		Rating:     ts.TeeSet.Rating,
		Holes:      holes,
	}
}

func toCourseDTO(c golf.Course) sdk.Course {
	return sdk.Course{ID: c.ID, Name: c.Name}
}

func toCourseTeeSetDTO(ts golf.CourseTeeSet) sdk.TeeSetSummary {
	return sdk.TeeSetSummary{
		CourseID:   ts.CourseID,
		TeeColorID: ts.TeeColorID,
		Color:      ts.Color,
		Slope:      ts.Slope,
		Rating:     ts.Rating,
	}
}

// dateString formats a date as YYYY-MM-DD, or "" if unset.
func dateString(d time.Time) string {
	if d.IsZero() {
		return ""
	}
	return d.Format("2006-01-02")
}

// parseDate parses a YYYY-MM-DD string into a UTC time.Time.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// pathUUID parses a UUID path parameter.
func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue(name))
}
