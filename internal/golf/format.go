package golf

import "fmt"

// FormatHoleStatus renders a HoleResult's match-play margin as text ("AS",
// "2 UP", "3 & 2"). This is presentation only — and note it renders only the
// margin, not the colour or name of the leader; who leads is HoleResult.LeaderTeamID,
// which the caller maps to a team. Kept separate from the scoring domain so the
// logic never depends on — or parses — a display string.
func FormatHoleStatus(h HoleResult) string {
	if h.Lead == 0 {
		return "AS"
	}
	if h.Decided {
		return fmt.Sprintf("%d & %d", h.Lead, h.HolesRemaining)
	}
	return fmt.Sprintf("%d UP", h.Lead)
}
