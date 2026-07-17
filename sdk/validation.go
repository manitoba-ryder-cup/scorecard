package sdk

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DateFormat is the wire format for date-only fields (ISO-8601 calendar date).
const DateFormat = "2006-01-02"

// Request-shape validation lives here, on the SDK types, so it is defined once and
// invoked at every boundary — the client (before sending), and each server transport
// (REST now, gRPC later). Only context-free checks belong here; rules that need
// stored state (does this exist, is it already taken) are domain invariants.

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
	return validateRequired(r.Color, "color")
}

// Validate checks a course creation request.
func (r CreateCourseRequest) Validate(ctx context.Context) error {
	return validateRequired(r.Name, "name")
}

// maxTierLen matches the tier column's VARCHAR(32).
const maxTierLen = 32

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
	if len(r.Tier) > maxTierLen {
		return fmt.Errorf("tier must be at most %d characters", maxTierLen)
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

// Validate checks a player creation request. Email and user_id are optional; an
// email, when given, must be well-formed.
func (r CreatePlayerRequest) Validate(ctx context.Context) error {
	if err := validateRequired(r.FirstName, "first_name"); err != nil {
		return err
	}
	if err := validateRequired(r.LastName, "last_name"); err != nil {
		return err
	}
	if r.Email != nil {
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
	if r.Strokes < 1 {
		return fmt.Errorf("strokes must be positive")
	}
	return nil
}
