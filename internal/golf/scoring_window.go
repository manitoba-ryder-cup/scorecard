package golf

import (
	"time"
	// Embed the timezone database: the runtime image carries tzdata today, but a missing
	// zone silently reads as UTC, which shifts the day a cup can be scored on.
	_ "time/tzdata"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// scoringOpen reports whether a tournament's scores can be recorded at the given moment:
// on any of its days, read where the golf is played. A tournament's dates are calendar
// dates, and a calendar date only means something somewhere — read as UTC, a Manitoba
// cup's final day ends at 19:00 local and cuts off a last group that tees off in the
// afternoon and finishes past midnight UTC.
//
// Deliberately not tied to a tee time: those move for weather, and aren't consistently
// stored as absolute instants across imported events.
func scoringOpen(now time.Time, t *Tournament) bool {
	today := dayIn(now.In(eventLocation(t.TimeZone)))
	return !today.Before(dayIn(t.StartDate)) && !today.After(dayIn(t.EndDate))
}

// eventLocation resolves a tournament's IANA zone, falling back to where the cup has
// always been played rather than to UTC — UTC would quietly close the window early on
// the final day, which is the failure this whole thing exists to avoid.
func eventLocation(name string) *time.Location {
	if name == "" {
		name = sdk.DefaultTimeZone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc, err = time.LoadLocation(sdk.DefaultTimeZone)
		if err != nil {
			return time.UTC
		}
	}
	return loc
}

// dayIn strips a timestamp to its calendar date, so two dates compare by day alone.
func dayIn(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
