package golf

import "fmt"

// FormatHoleStatus renders a HoleResult's match-play state as text ("AS",
// "2 UP", "3 & 2"). This is presentation only, kept separate from the scoring
// domain so the logic never depends on — or parses — a display string.
func FormatHoleStatus(h HoleResult) string {
	if h.Lead == 0 {
		return "AS"
	}
	lead := abs(h.Lead)
	if h.Decided {
		return fmt.Sprintf("%d & %d", lead, h.HolesRemaining)
	}
	return fmt.Sprintf("%d UP", lead)
}
