package rest

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// Handlers must never serialize a domain struct: the SDK's shape is the contract.

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
		Tier:             h.Tier,
		Biography:        h.Biography,
	}
}

func toPairRecordDTO(r golf.PairRecord) sdk.PairRecord {
	return sdk.PairRecord{
		PlayerID:  r.PlayerID,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Matches:   r.Matches,
		Record:    toPlayerRecordDTO(r.Record),
	}
}

func toNotableMatchDTO(m *golf.NotableMatch) *sdk.NotableMatch {
	if m == nil {
		return nil
	}
	return &sdk.NotableMatch{Year: m.Year, Lead: m.Lead, HolesRemaining: m.HolesRemaining, Opponents: m.Opponents}
}

func toPlayerStatsDTO(s golf.PlayerStats) sdk.PlayerStats {
	return sdk.PlayerStats{
		ByFormat: mapSlice(s.ByFormat, func(f golf.FormatRecord) sdk.FormatRecord {
			return sdk.FormatRecord{FormatName: f.FormatName, Record: toPlayerRecordDTO(f.Record)}
		}),
		Teammates:    mapSlice(s.Teammates, toPairRecordDTO),
		Opponents:    mapSlice(s.Opponents, toPairRecordDTO),
		Points:       s.Points,
		CupsPlayed:   s.CupsPlayed,
		LastHole:     toPlayerRecordDTO(s.LastHole),
		DecidedEarly: toPlayerRecordDTO(s.DecidedEarly),
		BestWin:      toNotableMatchDTO(s.BestWin),
		HeaviestLoss: toNotableMatchDTO(s.HeaviestLoss),
	}
}

func toTournamentDTO(t golf.Tournament) sdk.Tournament {
	return sdk.Tournament{
		ID:        t.ID,
		Name:      t.Name,
		StartDate: dateString(t.StartDate),
		EndDate:   dateString(t.EndDate),
		Location:  t.Location,
		Phase:     t.Phase,
	}
}

func toTournamentTeamDTO(td golf.TeamData) sdk.TournamentTeam {
	var captain *sdk.PlayerSummary
	if td.Captain != nil {
		captain = &sdk.PlayerSummary{
			ID:        td.Captain.ID,
			FirstName: td.Captain.FirstName,
			LastName:  td.Captain.LastName,
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
	return sdk.HoleStatus{
		HoleNumber:     h.HoleNumber,
		TeamScores:     mapSlice(h.TeamScores, toTeamHoleScoreDTO),
		LeaderTeamID:   h.LeaderTeamID,
		Lead:           h.Lead,
		HolesRemaining: h.HolesRemaining,
		Decided:        h.Decided,
	}
}

// toMatchStatusDTO surfaces the winner explicitly rather than leaving every client to
// re-encode "the leader is the winner once it's finished".
func toMatchStatusDTO(r golf.StoredResult) sdk.MatchStatus {
	return sdk.MatchStatus{
		Finished:       r.Finished,
		WinnerTeamID:   r.Winner(),
		LeaderTeamID:   r.LeaderTeamID,
		Lead:           r.Lead,
		HolesRemaining: r.HolesRemaining,
	}
}

func toTeamHoleScoreDTO(t golf.TeamHoleScore) sdk.TeamHoleScore {
	return sdk.TeamHoleScore{
		TeamID:       t.TeamID,
		Strokes:      t.Strokes,
		PlayerScores: mapSlice(t.PlayerScores, toPlayerHoleScoreDTO),
	}
}

func toPlayerHoleScoreDTO(p golf.PlayerHoleScore) sdk.PlayerHoleScore {
	return sdk.PlayerHoleScore{PlayerID: p.PlayerID, Strokes: p.Strokes}
}

func toHoleDTO(h golf.Hole) sdk.Hole {
	return sdk.Hole{Number: h.Number, Par: h.Par, Hdcp: h.Hdcp, Yards: h.Yards}
}

func toMatchResultDTO(m golf.MatchResult) sdk.MatchResult {
	// Empty (not null) so the client always gets an array to iterate.
	holeResults := m.HoleResults
	if holeResults == nil {
		holeResults = []*uuid.UUID{}
	}
	opens, closes := golf.ScoringWindow(m.TeeTime)
	return sdk.MatchResult{
		MatchStatus:     toMatchStatusDTO(m.StoredResult),
		MatchID:         m.MatchID,
		FormatName:      m.FormatName,
		PlayersPerSide:  m.PlayersPerSide,
		ScoresPerPlayer: m.ScoresPerPlayer,
		Sides:           mapSlice(m.Sides, toMatchSideDTO),
		HoleResults:     holeResults,
		TeeTime:         m.TeeTime.Format(time.RFC3339),
		CourseName:      m.CourseName,
		ScoringOpensAt:  opens.Format(time.RFC3339),
		ScoringClosesAt: closes.Format(time.RFC3339),
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
	return sdk.Course{ID: c.ID, Name: c.Name, TimeZone: c.TimeZone}
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

// pathUUIDOr400 parses a UUID path parameter, writing a 400 and returning ok=false when
// it is malformed. resource names the thing in the message ("tournament" -> "Invalid
// tournament ID").
func pathUUIDOr400(w http.ResponseWriter, req *http.Request, name, resource string) (uuid.UUID, bool) {
	id, err := uuid.Parse(req.PathValue(name))
	if err != nil {
		respondError(req.Context(), w, http.StatusBadRequest, "Invalid "+resource+" ID", err)
		return uuid.Nil, false
	}
	return id, true
}
