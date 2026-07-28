package golf

import (
	"testing"
	"time"
)

// The 2026 cup: two days at Buffalo Point, last group off at 15:50 local.
var (
	cupStart = time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	cupEnd   = time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)
)

func cup() *Tournament {
	return &Tournament{Name: "Manitoba Ryder Cup", StartDate: cupStart, EndDate: cupEnd}
}

func TestScoringOpen(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"months before the cup", time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), false},
		{"the day before", time.Date(2026, 9, 17, 18, 0, 0, 0, time.UTC), false},
		{"first morning", time.Date(2026, 9, 18, 13, 0, 0, 0, time.UTC), true},
		{"second day", time.Date(2026, 9, 19, 15, 0, 0, 0, time.UTC), true},
		// The reason this is read in the event's own timezone: the last group tees off at
		// 15:50 local and finishes past midnight UTC. Read as UTC dates they would be cut
		// off around the 14th hole of the cup's final match.
		{"last group still out, after midnight UTC", time.Date(2026, 9, 20, 1, 20, 0, 0, time.UTC), true},
		{"a nightcap at 11pm local on the last day", time.Date(2026, 9, 20, 4, 0, 0, 0, time.UTC), true},
		{"the morning after", time.Date(2026, 9, 20, 14, 0, 0, 0, time.UTC), false},
		// Just before midnight local on the eve — still not the day of.
		{"11pm local the night before", time.Date(2026, 9, 18, 4, 0, 0, 0, time.UTC), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scoringOpen(tc.now, cup(), "America/Winnipeg"); got != tc.want {
				t.Errorf("scoringOpen(%s) = %v, want %v", tc.now.Format(time.RFC3339), got, tc.want)
			}
		})
	}
}

func TestScoringOpen_ReadsTheCoursesZone(t *testing.T) {
	// Same instant, same dates: 22:00 on the 19th in Winnipeg is already the 20th in
	// Auckland, so a round played there is over while the Manitoba one is still going.
	instant := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)

	if !scoringOpen(instant, cup(), "America/Winnipeg") {
		t.Error("Winnipeg: want open, it is still 22:00 on the final day there")
	}
	if scoringOpen(instant, cup(), "Pacific/Auckland") {
		t.Error("Auckland: want shut, that course's last day ended hours ago")
	}
}

func TestScoringOpen_FallsBackToTheCupsHomeZone(t *testing.T) {
	// A tournament stored before zones were recorded, or with an unreadable one, must not
	// quietly become UTC — that closes the window early on the final day.
	lastGroupOut := time.Date(2026, 9, 20, 1, 20, 0, 0, time.UTC)

	for _, zone := range []string{"", "Not/AZone"} {
		if !scoringOpen(lastGroupOut, cup(), zone) {
			t.Errorf("zone %q: want the last group still able to score", zone)
		}
	}
}
