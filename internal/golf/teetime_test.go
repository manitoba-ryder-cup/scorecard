package golf

import (
	"errors"
	"testing"
	"time"
)

func TestParseTeeTime(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		timeZone string
		want     string // RFC3339 in UTC; "" means an error is expected
	}{
		{"a wall clock is read at the course", "2026-09-18T08:00", "America/Winnipeg", "2026-09-18T13:00:00Z"},
		{"seconds are optional but accepted", "2026-09-18T08:00:30", "America/Winnipeg", "2026-09-18T13:00:30Z"},
		// Winnipeg is UTC-5 in September (CDT) and UTC-6 in November (CST). A fixed
		// offset instead of a zone would put one of these two an hour out.
		{"the zone's own offset applies, not a fixed one", "2026-11-18T08:00", "America/Winnipeg", "2026-11-18T14:00:00Z"},
		{"a course that never observes DST", "2026-09-18T08:00", "America/Phoenix", "2026-09-18T15:00:00Z"},
		{"an explicit instant is trusted as given", "2026-09-18T13:00:00Z", "America/Winnipeg", "2026-09-18T13:00:00Z"},
		{"so is an offset from somewhere else", "2026-09-18T08:00:00-04:00", "America/Winnipeg", "2026-09-18T12:00:00Z"},
		// LoadLocation("") is UTC, which would shift every tee time by the course's
		// offset instead of failing anywhere a person would see it.
		{"an unset zone falls back to the cup's", "2026-09-18T08:00", "", "2026-09-18T13:00:00Z"},
		{"a date alone is not a tee time", "2026-09-18", "America/Winnipeg", ""},
		{"gibberish is rejected", "half eight", "America/Winnipeg", ""},
		{"an unloadable zone is rejected", "2026-09-18T08:00", "Mars/Olympus_Mons", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTeeTime(tc.in, tc.timeZone)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("ParseTeeTime(%q, %q) = %s, want an error", tc.in, tc.timeZone, got.Format(time.RFC3339))
				}
				// The API maps this sentinel to 400; anything else answers 500.
				if !errors.Is(err, ErrInvalidInput) {
					t.Errorf("error should wrap ErrInvalidInput, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTeeTime(%q, %q): %v", tc.in, tc.timeZone, err)
			}
			if gotUTC := got.UTC().Format(time.RFC3339); gotUTC != tc.want {
				t.Errorf("ParseTeeTime(%q, %q) = %s, want %s", tc.in, tc.timeZone, gotUTC, tc.want)
			}
		})
	}
}
