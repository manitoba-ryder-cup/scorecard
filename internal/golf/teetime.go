package golf

import (
	"fmt"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// ParseTeeTime reads a tee time as either an instant or a wall clock. A bare wall clock is
// what a tee sheet actually says, so it is read at the course being played — writing those
// as UTC is how thirteen years of history ended up teeing off at 3am. An explicit offset is
// trusted as given, so a caller that already holds an instant is unaffected.
//
// An empty timeZone falls back to the cup's rather than to LoadLocation's UTC, which would
// silently shift every tee time by the course's offset instead of failing visibly.
func ParseTeeTime(s, timeZone string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if timeZone == "" {
		timeZone = sdk.DefaultTimeZone
	}
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: course time zone %q is not an IANA name", ErrInvalidInput, timeZone)
	}
	for _, layout := range sdk.WallClockLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: invalid tee time %q: want a wall clock (2006-01-02T15:04) or an RFC3339 instant", ErrInvalidInput, s)
}
