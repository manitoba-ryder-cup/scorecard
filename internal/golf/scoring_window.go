package golf

import (
	"time"
	// Embed the timezone database: the runtime image carries tzdata today, but the
	// scoring window silently mis-reads if it ever doesn't.
	_ "time/tzdata"
)

// eventLocation is where the cup is played. A tournament's start and end are calendar
// dates, and a calendar date only means anything somewhere — read as UTC, the final
// day ends at 19:00 local, cutting off the last group mid-round, because an afternoon
// tee time in Manitoba finishes after midnight UTC.
var eventLocation = mustLoadLocation("America/Winnipeg")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Falling back to UTC would quietly close the window early on the last day;
		// better to fail at startup than mid-tournament.
		panic("golf: cannot load event timezone " + name + ": " + err.Error())
	}
	return loc
}

// scoringOpen reports whether a tournament's scores can be recorded at the given moment:
// on any of its days, read where the golf is played. Not tied to a tee time, which moves
// for weather and is not consistently stored as an absolute instant.
func scoringOpen(now, startDate, endDate time.Time) bool {
	today := dayIn(now.In(eventLocation))
	return !today.Before(dayIn(startDate)) && !today.After(dayIn(endDate))
}

// dayIn strips a timestamp to its calendar date, so two dates compare by day alone.
func dayIn(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
