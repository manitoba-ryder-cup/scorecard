package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// SeedInput is the contract for the advance setup of a tournament: the event, its roster
// (with per-tournament tiers), the two captains, and the match schedule. The draft (which
// player is on which team) and assigning players to matches happen live at the event, so
// this seeds neither — only the captains get a team. Course, tee color, and formats are
// referenced by name and must already exist. Players are matched by email (created only
// if new), so a player recurs across tournaments instead of being duplicated.
type SeedInput struct {
	Tournament SeedTournamentMeta `json:"tournament"`
	Course     string             `json:"course"`
	TeeColor   string             `json:"tee_color"`
	// Captains maps a team colour ("Red"/"Blue") to the captain's email, which must be
	// one of the entered players.
	Captains map[string]string `json:"captains"`
	// Players is the entered roster. Team assignment is left to the live draft.
	Players []SeedPlayer     `json:"players"`
	Matches []SeedMatchGroup `json:"matches"`
}

type SeedTournamentMeta struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date"`
	Location  string `json:"location"`
}

type SeedPlayer struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Tier      string  `json:"tier"`
	Biography string  `json:"biography"`
	Hdcp      float32 `json:"hdcp"`
}

type SeedMatchGroup struct {
	Format string `json:"format"`
	// A tee sheet's own wall clock ("2026-09-18T08:00"), read in the tournament's zone.
	// An explicit offset is honoured as given.
	TeeTimes []string `json:"tee_times"`
}

// SeedSummary reports what a seed run created.
type SeedSummary struct {
	TournamentID   uuid.UUID
	PlayersEntered int
	Matches        int
}

// SeedTournament validates the setup and writes it.
//
// Planning resolves every name and parses every time without touching the database, so a
// typo in a hand-edited file costs nothing. Writing is one call, and one transaction: the
// event either exists in full or not at all. Neither half is redundant — planning turns a
// bad file into a free error, and the transaction covers what planning cannot see, like
// the connection dropping at match ten.
func SeedTournament(ctx context.Context, svc *Services, in *SeedInput) (*SeedSummary, error) {
	plan, err := planSeed(ctx, svc, in)
	if err != nil {
		return nil, err
	}
	summary, err := svc.Seed.Seed(ctx, *plan)
	if err != nil {
		return nil, err
	}
	return &SeedSummary{
		TournamentID:   summary.TournamentID,
		PlayersEntered: summary.PlayersEntered,
		Matches:        summary.Matches,
	}, nil
}

// planSeed resolves and checks the whole file without writing anything. Every error it
// returns is one the caller can fix by editing the file and running again, against a
// database it has not touched.
func planSeed(ctx context.Context, svc *Services, in *SeedInput) (*golf.SeedPlan, error) {
	course, err := lookupCourse(ctx, svc, in.Course)
	if err != nil {
		return nil, err
	}
	teeColorID, err := lookupTeeColor(ctx, svc, in.TeeColor)
	if err != nil {
		return nil, err
	}
	formatIDs, err := lookupFormats(ctx, svc)
	if err != nil {
		return nil, err
	}

	start, err := time.Parse(time.DateOnly, in.Tournament.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid tournament start_date: %w", err)
	}
	end, err := time.Parse(time.DateOnly, in.Tournament.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid tournament end_date: %w", err)
	}

	players, roster, err := planRoster(in.Players)
	if err != nil {
		return nil, err
	}
	captains, err := planCaptains(in.Captains, roster)
	if err != nil {
		return nil, err
	}

	var matches []golf.PlannedMatch
	for _, mg := range in.Matches {
		formatID, ok := formatIDs[mg.Format]
		if !ok {
			return nil, fmt.Errorf("unknown match format %q", mg.Format)
		}
		for _, tt := range mg.TeeTimes {
			teeTime, err := parseTeeTime(tt, course.TimeZone)
			if err != nil {
				return nil, err
			}
			matches = append(matches, golf.PlannedMatch{Format: mg.Format, FormatID: formatID, TeeTime: teeTime})
		}
	}

	return &golf.SeedPlan{
		Tournament: golf.CreateTournamentInput{
			Name: in.Tournament.Name, StartDate: start, EndDate: end, Location: in.Tournament.Location,
		},
		CourseID:   course.ID,
		TeeColorID: teeColorID,
		Players:    players,
		Captains:   captains,
		Matches:    matches,
	}, nil
}

// planRoster returns the roster in file order and indexed by lowercased email. A player
// without one cannot be recognised next year, and a repeated one would be entered twice
// under the same identity. Tier is defaulted here, so the plan is exactly what lands.
func planRoster(in []SeedPlayer) ([]golf.SeedPlayer, map[string]golf.SeedPlayer, error) {
	players := make([]golf.SeedPlayer, 0, len(in))
	byEmail := make(map[string]golf.SeedPlayer, len(in))
	for _, sp := range in {
		if strings.TrimSpace(sp.Email) == "" {
			return nil, nil, fmt.Errorf("player %s %s has no email", sp.FirstName, sp.LastName)
		}
		key := strings.ToLower(strings.TrimSpace(sp.Email))
		if _, dup := byEmail[key]; dup {
			return nil, nil, fmt.Errorf("player %s appears twice in the roster", sp.Email)
		}
		p := golf.SeedPlayer{
			FirstName: sp.FirstName, LastName: sp.LastName, Email: sp.Email,
			Tier: golf.TierOrDefault(sp.Tier), Biography: sp.Biography, Hdcp: sp.Hdcp,
		}
		players = append(players, p)
		byEmail[key] = p
	}
	return players, byEmail, nil
}

// planCaptains checks each captain names a real side and someone actually entered.
func planCaptains(captains map[string]string, roster map[string]golf.SeedPlayer) (map[string]string, error) {
	out := make(map[string]string, len(captains))
	for color, email := range captains {
		if color != sdk.TeamColorRed && color != sdk.TeamColorBlue {
			return nil, fmt.Errorf("captain colour %q is not %s or %s", color, sdk.TeamColorRed, sdk.TeamColorBlue)
		}
		key := strings.ToLower(strings.TrimSpace(email))
		if _, ok := roster[key]; !ok {
			return nil, fmt.Errorf("%s captain %q is not in the roster", color, email)
		}
		out[color] = key
	}
	return out, nil
}

// lookupCourse returns the whole course, not just its id: its timezone is what a bare
// tee time in the seed file is read against.
func lookupCourse(ctx context.Context, svc *Services, name string) (golf.Course, error) {
	courses, err := svc.Course.ListCourses(ctx)
	if err != nil {
		return golf.Course{}, fmt.Errorf("listing courses: %w", err)
	}
	for _, c := range courses {
		if c.Name == name {
			return c, nil
		}
	}
	return golf.Course{}, fmt.Errorf("course %q not found (create it first)", name)
}

func lookupTeeColor(ctx context.Context, svc *Services, color string) (uuid.UUID, error) {
	colors, err := svc.Course.ListTeeColors(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("listing tee colors: %w", err)
	}
	for _, tc := range colors {
		if tc.Color == color {
			return tc.ID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("tee color %q not found (create it first)", color)
}

func lookupFormats(ctx context.Context, svc *Services) (map[string]uuid.UUID, error) {
	formats, err := svc.Format.ListFormats(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing formats: %w", err)
	}
	m := make(map[string]uuid.UUID, len(formats))
	for _, f := range formats {
		m[f.Name] = f.ID
	}
	return m, nil
}

// parseTeeTime reads a tee time from a seed file. A bare wall clock is what a tee sheet
// actually says, so it is read at the course being played — writing those as UTC is how
// thirteen years of history ended up teeing off at 3am. An explicit offset is trusted as
// given, so a file that already carries one is unaffected.
func parseTeeTime(s, timeZone string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return time.Time{}, fmt.Errorf("course time_zone %q is not an IANA name: %w", timeZone, err)
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid tee_time %q: want a wall clock (2006-01-02T15:04) or an RFC3339 instant", s)
}
