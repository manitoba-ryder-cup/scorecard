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
			if got := scoringOpen(tc.now, cupStart, cupEnd); got != tc.want {
				t.Errorf("scoringOpen(%s) = %v, want %v", tc.now.Format(time.RFC3339), got, tc.want)
			}
		})
	}
}
