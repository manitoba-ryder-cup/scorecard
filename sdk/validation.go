package sdk

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	// Embedded so an IANA name validates the same wherever the SDK runs.
	_ "time/tzdata"

	"github.com/google/uuid"
)

// DateFormat is the wire format for date-only fields (ISO-8601 calendar date).
const DateFormat = "2006-01-02"

// On the SDK types so it runs at every boundary; stored-state rules are domain invariants.

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateRequired(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func validateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// validateDate confirms a required date field is present and parses as YYYY-MM-DD,
// returning the parsed value so callers can compare dates.
func validateDate(value, field string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("%s is required", field)
	}
	t, err := time.Parse(DateFormat, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be a valid date (YYYY-MM-DD)", field)
	}
	return t, nil
}

// Validate checks a tee-color creation request.
func (r CreateTeeColorRequest) Validate(ctx context.Context) error {
	if err := validateRequired(r.Color, "color"); err != nil {
		return err
	}
	return validateMaxLen(r.Color, "color", maxColorLen)
}

// Validate checks a course creation request.
func (r CreateCourseRequest) Validate(ctx context.Context) error {
	if err := validateRequired(r.Name, "name"); err != nil {
		return err
	}
	if err := validateMaxLen(r.Name, "name", maxTitleLen); err != nil {
		return err
	}
	// An unreadable zone resolves to UTC and silently shifts every round at that course.
	if r.TimeZone != "" {
		if _, err := time.LoadLocation(r.TimeZone); err != nil {
			return fmt.Errorf("time_zone must be an IANA name such as America/Winnipeg")
		}
	}
	return nil
}

// Length caps mirror the schema's VARCHAR widths, so an over-long value is a client
// error at the boundary instead of a truncation error from Postgres.
const (
	maxTierLen     = 32
	maxNameLen     = 32  // players.first_name / last_name
	maxColorLen    = 32  // tee_colors.color
	maxTitleLen    = 255 // courses.name, tournaments.name
	maxLocationLen = 255 // tournaments.location
	maxEmailLen    = 255 // players.email
)

func validateMaxLen(value, field string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%s must be at most %d characters", field, max)
	}
	return nil
}

// Validate checks a tournament-entry request: a player reference and a tier that fits.
func (r EnterTournamentPlayerRequest) Validate(ctx context.Context) error {
	if r.PlayerID == uuid.Nil {
		return fmt.Errorf("player_id is required")
	}
	if len(r.Tier) > maxTierLen {
		return fmt.Errorf("tier must be at most %d characters", maxTierLen)
	}
	return nil
}

// Validate checks a tournament-entry update. Handicap may be negative (plus handicaps).
func (r UpdateTournamentPlayerRequest) Validate(ctx context.Context) error {
	if r.Tier == nil && r.Biography == nil && r.Hdcp == nil {
		return fmt.Errorf("no fields to update")
	}
	if r.Tier != nil {
		// Blanking a tier would leave an entry with no flight, which entering one cannot do.
		if strings.TrimSpace(*r.Tier) == "" {
			return fmt.Errorf("tier must not be empty")
		}
		if len(*r.Tier) > maxTierLen {
			return fmt.Errorf("tier must be at most %d characters", maxTierLen)
		}
	}
	return nil
}

// Validate checks a match creation request: the three references are required, and
// tee_time (when given) must be RFC3339.
func (r CreateMatchRequest) Validate(ctx context.Context) error {
	if r.CourseID == uuid.Nil {
		return fmt.Errorf("course_id is required")
	}
	if r.TeeColorID == uuid.Nil {
		return fmt.Errorf("tee_color_id is required")
	}
	if r.MatchFormatID == uuid.Nil {
		return fmt.Errorf("match_format_id is required")
	}
	// The scoring window is measured from the tee time, so one without it cannot be scored.
	if strings.TrimSpace(r.TeeTime) == "" {
		return fmt.Errorf("tee_time is required")
	}
	if _, err := time.Parse(time.RFC3339, r.TeeTime); err != nil {
		return fmt.Errorf("tee_time must be RFC3339 (e.g. 2026-08-01T08:00:00Z)")
	}
	return nil
}

// Validate checks a match update. A present field must be usable — an explicit null uuid
// or a blank tee time is a caller error, not a request to leave the value alone.
func (r UpdateMatchRequest) Validate(ctx context.Context) error {
	if r.CourseID == nil && r.TeeColorID == nil && r.TeeTime == nil && r.Handicapped == nil {
		return fmt.Errorf("no fields to update")
	}
	if r.CourseID != nil && *r.CourseID == uuid.Nil {
		return fmt.Errorf("course_id must not be empty")
	}
	if r.TeeColorID != nil && *r.TeeColorID == uuid.Nil {
		return fmt.Errorf("tee_color_id must not be empty")
	}
	if r.TeeTime != nil {
		if _, err := time.Parse(time.RFC3339, *r.TeeTime); err != nil {
			return fmt.Errorf("tee_time must be RFC3339 (e.g. 2026-08-01T08:00:00Z)")
		}
	}
	return nil
}

// Validate checks a lineup submission's shape. How many a side belongs to the match format,
// which only the server knows, so this checks what a caller can get wrong on its own.
func (r SetLineupRequest) Validate(ctx context.Context) error {
	if len(r.Participants) == 0 {
		return fmt.Errorf("participants is required")
	}
	seen := make(map[uuid.UUID]bool, len(r.Participants))
	for i, p := range r.Participants {
		if p.PlayerID == uuid.Nil {
			return fmt.Errorf("participants[%d].player_id is required", i)
		}
		if p.TeamID == uuid.Nil {
			return fmt.Errorf("participants[%d].team_id is required", i)
		}
		if seen[p.PlayerID] {
			return fmt.Errorf("participants[%d] names a player already in the lineup", i)
		}
		seen[p.PlayerID] = true
	}
	return nil
}

// Validate checks a draft request: a player reference is required.
func (r DraftPlayerRequest) Validate(ctx context.Context) error {
	if r.PlayerID == uuid.Nil {
		return fmt.Errorf("player_id is required")
	}
	return nil
}

// Validate checks a set-captain request.
func (r SetTeamCaptainRequest) Validate(ctx context.Context) error {
	if r.CaptainID == uuid.Nil {
		return fmt.Errorf("captain_id is required")
	}
	return nil
}

// Validate checks a tee-set creation request: valid slope/rating and exactly 18
// holes forming complete, non-duplicated hole-number and stroke-index (hdcp) sets.
func (r CreateTeeSetRequest) Validate(ctx context.Context) error {
	if r.TeeColorID == uuid.Nil {
		return fmt.Errorf("tee_color_id is required")
	}
	if r.Slope < 55 || r.Slope > 155 {
		return fmt.Errorf("slope must be between 55 and 155")
	}
	if r.Rating <= 0 {
		return fmt.Errorf("rating must be positive")
	}
	if len(r.Holes) != 18 {
		return fmt.Errorf("exactly 18 holes are required, got %d", len(r.Holes))
	}
	seenNumber := make(map[int32]bool, 18)
	seenHdcp := make(map[int32]bool, 18)
	for _, h := range r.Holes {
		if h.Number < 1 || h.Number > 18 {
			return fmt.Errorf("hole number must be between 1 and 18")
		}
		if seenNumber[h.Number] {
			return fmt.Errorf("duplicate hole number %d", h.Number)
		}
		seenNumber[h.Number] = true
		if h.Par < 3 || h.Par > 6 {
			return fmt.Errorf("hole %d: par must be between 3 and 6", h.Number)
		}
		if h.Hdcp < 1 || h.Hdcp > 18 {
			return fmt.Errorf("hole %d: hdcp must be between 1 and 18", h.Number)
		}
		if seenHdcp[h.Hdcp] {
			return fmt.Errorf("duplicate hdcp (stroke index) %d", h.Hdcp)
		}
		seenHdcp[h.Hdcp] = true
		if h.Yards <= 0 {
			return fmt.Errorf("hole %d: yards must be positive", h.Number)
		}
	}
	return nil
}

func (r UpdatePlayerRequest) Validate(ctx context.Context) error {
	if r.FirstName == nil && r.LastName == nil && r.Email == nil && r.PhotoPath == nil {
		return fmt.Errorf("at least one field must be set")
	}
	// A name can be corrected but not removed; clearing a photo is how one is taken down.
	if r.FirstName != nil {
		if err := validateRequired(*r.FirstName, "first_name"); err != nil {
			return err
		}
		if err := validateMaxLen(*r.FirstName, "first_name", maxNameLen); err != nil {
			return err
		}
	}
	if r.LastName != nil {
		if err := validateRequired(*r.LastName, "last_name"); err != nil {
			return err
		}
		if err := validateMaxLen(*r.LastName, "last_name", maxNameLen); err != nil {
			return err
		}
	}
	if r.Email != nil {
		if err := validateRequired(*r.Email, "email"); err != nil {
			return err
		}
		if err := validateEmail(*r.Email); err != nil {
			return err
		}
	}
	if r.PhotoPath != nil {
		if err := validateMaxLen(*r.PhotoPath, "photo_path", maxNameLen); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks a player creation request. Email and user_id are optional; an
// email, when given, must be well-formed.
func (r CreatePlayerRequest) Validate(ctx context.Context) error {
	if err := validateRequired(r.FirstName, "first_name"); err != nil {
		return err
	}
	if err := validateMaxLen(r.FirstName, "first_name", maxNameLen); err != nil {
		return err
	}
	if err := validateRequired(r.LastName, "last_name"); err != nil {
		return err
	}
	if err := validateMaxLen(r.LastName, "last_name", maxNameLen); err != nil {
		return err
	}
	if r.Email != nil {
		if err := validateMaxLen(*r.Email, "email", maxEmailLen); err != nil {
			return err
		}
		if err := validateEmail(*r.Email); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks a tournament creation request: name present, both dates valid, and
// the end not before the start.
func (r CreateTournamentRequest) Validate(ctx context.Context) error {
	if err := validateRequired(r.Name, "name"); err != nil {
		return err
	}
	if err := validateMaxLen(r.Name, "name", maxTitleLen); err != nil {
		return err
	}
	if err := validateMaxLen(r.Location, "location", maxLocationLen); err != nil {
		return err
	}
	start, err := validateDate(r.StartDate, "start_date")
	if err != nil {
		return err
	}
	end, err := validateDate(r.EndDate, "end_date")
	if err != nil {
		return err
	}
	if end.Before(start) {
		return fmt.Errorf("end_date cannot precede start_date")
	}
	return nil
}

// Validate checks a score submission's shape. Which team/player the score belongs to
// (must be in the match) is a domain invariant, checked server-side.
func (r ScoreSubmission) Validate(ctx context.Context) error {
	if r.HoleNumber < 1 || r.HoleNumber > 18 {
		return fmt.Errorf("hole_number must be between 1 and 18")
	}
	if len(r.Scores) == 0 {
		return fmt.Errorf("scores must not be empty")
	}
	// Two scores for one competitor would resolve to whichever landed last, so it is refused.
	seen := make(map[uuid.UUID]bool, len(r.Scores))
	for _, s := range r.Scores {
		if s.Strokes < 1 {
			return fmt.Errorf("strokes must be positive")
		}
		subject := s.TeamID
		if s.PlayerID != nil {
			subject = *s.PlayerID
		}
		if seen[subject] {
			return fmt.Errorf("duplicate score for the same team or player")
		}
		seen[subject] = true
	}
	return nil
}
