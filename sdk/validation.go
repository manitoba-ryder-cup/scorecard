package sdk

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
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
