package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
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

// SeedTournament creates a tournament (with its two teams), enters the roster, and marks
// each side's captain (drafting only the captain onto their team). It then creates the
// matches for each format, leaving them without participants. The field's draft and match
// participants are assigned live at the event, not here. The course, tee color, and
// formats are referenced by name and must already exist.
//
// The whole seed is one transaction, and everything the file says is resolved before the
// first write. Either together would be enough for a typo; both are here because they
// fail differently — planning turns a bad file into an error that costs nothing, and the
// transaction covers what planning cannot see, like the connection dropping at match ten.
func SeedTournament(ctx context.Context, svc *Services, in *SeedInput) (*SeedSummary, error) {
	plan, err := planSeed(ctx, svc, in)
	if err != nil {
		return nil, err
	}

	var summary *SeedSummary
	err = postgres.WithinTenantTx(ctx, svc.db, func(ctx context.Context) error {
		summary, err = applySeed(ctx, svc, plan)
		return err
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

// seedPlan is a validated seed: every name resolved, every time parsed, every cross
// reference checked. Nothing in it can fail for a reason the file could have told us.
type seedPlan struct {
	tournament golf.CreateTournamentInput
	course     golf.Course
	teeColorID uuid.UUID
	players    []SeedPlayer
	captains   map[string]string // team colour -> lowercased captain email
	matches    []plannedMatch
}

// plannedMatch is one match with its format resolved and its tee time already an instant.
type plannedMatch struct {
	format   string
	formatID uuid.UUID
	teeTime  time.Time
}

// planSeed resolves and checks the whole file without writing anything. Every error it
// returns is one the caller can fix by editing the file and running again, against a
// database it has not touched.
func planSeed(ctx context.Context, svc *Services, in *SeedInput) (*seedPlan, error) {
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

	roster, err := planRoster(in.Players)
	if err != nil {
		return nil, err
	}
	captains, err := planCaptains(in.Captains, roster)
	if err != nil {
		return nil, err
	}

	var matches []plannedMatch
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
			matches = append(matches, plannedMatch{format: mg.Format, formatID: formatID, teeTime: teeTime})
		}
	}

	return &seedPlan{
		tournament: golf.CreateTournamentInput{
			Name: in.Tournament.Name, StartDate: start, EndDate: end, Location: in.Tournament.Location,
		},
		course:     course,
		teeColorID: teeColorID,
		players:    in.Players,
		captains:   captains,
		matches:    matches,
	}, nil
}

// planRoster returns the roster keyed by lowercased email. A player without one cannot be
// recognised next year, and a repeated one would be entered twice under the same identity.
func planRoster(players []SeedPlayer) (map[string]SeedPlayer, error) {
	roster := make(map[string]SeedPlayer, len(players))
	for _, sp := range players {
		if strings.TrimSpace(sp.Email) == "" {
			return nil, fmt.Errorf("player %s %s has no email", sp.FirstName, sp.LastName)
		}
		key := strings.ToLower(strings.TrimSpace(sp.Email))
		if _, dup := roster[key]; dup {
			return nil, fmt.Errorf("player %s appears twice in the roster", sp.Email)
		}
		roster[key] = sp
	}
	return roster, nil
}

// planCaptains checks each captain names a real side and someone actually entered.
func planCaptains(captains map[string]string, roster map[string]SeedPlayer) (map[string]string, error) {
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

// applySeed performs the writes. Its errors are infrastructure, not input — the plan has
// already ruled out everything the file could have got wrong.
func applySeed(ctx context.Context, svc *Services, plan *seedPlan) (*SeedSummary, error) {
	tournament, err := svc.Tournament.CreateTournament(ctx, plan.tournament)
	if err != nil {
		return nil, fmt.Errorf("creating tournament: %w", err)
	}

	// The tournament seeds its two teams; map colour -> id so captains land on the right side.
	teams, err := svc.Team.ListTeamsByTournament(ctx, tournament.ID)
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	teamByColor := make(map[string]uuid.UUID, len(teams))
	for _, t := range teams {
		teamByColor[t.Color] = t.ID
	}

	finder, err := newPlayerFinder(ctx, svc)
	if err != nil {
		return nil, err
	}

	summary := &SeedSummary{TournamentID: tournament.ID}

	// Enter the whole roster; record ids by email so captains can be resolved.
	enteredByEmail := make(map[string]uuid.UUID, len(plan.players))
	for _, sp := range plan.players {
		playerID, err := finder.findOrCreate(ctx, svc, sp)
		if err != nil {
			return nil, err
		}
		if _, err := svc.Roster.EnterPlayer(ctx, golf.EnterPlayerInput{
			TournamentID: tournament.ID, PlayerID: playerID,
			Tier: sp.Tier, Biography: sp.Biography, Hdcp: sp.Hdcp,
		}); err != nil {
			return nil, fmt.Errorf("entering %s: %w", sp.Email, err)
		}
		enteredByEmail[strings.ToLower(strings.TrimSpace(sp.Email))] = playerID
		summary.PlayersEntered++
	}

	// Draft each captain onto their team and set them as captain. The rest of the field
	// is entered but undrafted — the draft happens live.
	for color, email := range plan.captains {
		teamID, ok := teamByColor[color]
		if !ok {
			return nil, fmt.Errorf("tournament has no %q team", color)
		}
		captainID := enteredByEmail[email]
		if _, err := svc.Roster.DraftPlayer(ctx, teamID, captainID); err != nil {
			return nil, fmt.Errorf("drafting %s captain: %w", color, err)
		}
		if _, err := svc.Team.SetCaptain(ctx, teamID, captainID); err != nil {
			return nil, fmt.Errorf("setting %s captain: %w", color, err)
		}
	}

	// Matches in schedule order (no participants — assigned live).
	for _, m := range plan.matches {
		if _, err := svc.Match.CreateMatch(ctx, golf.CreateMatchInput{
			TournamentID: tournament.ID, CourseID: plan.course.ID, TeeColorID: plan.teeColorID,
			MatchFormatID: m.formatID, TeeTime: m.teeTime,
		}); err != nil {
			return nil, fmt.Errorf("creating %s match: %w", m.format, err)
		}
		summary.Matches++
	}
	return summary, nil
}

// playerFinder resolves seed players to existing players by email (created only if new),
// so a returning player isn't duplicated year to year.
type playerFinder struct {
	byEmail map[string]uuid.UUID
}

func newPlayerFinder(ctx context.Context, svc *Services) (*playerFinder, error) {
	players, err := svc.Player.ListPlayers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing players: %w", err)
	}
	byEmail := make(map[string]uuid.UUID, len(players))
	for _, p := range players {
		if p.Email != nil {
			byEmail[strings.ToLower(*p.Email)] = p.ID
		}
	}
	return &playerFinder{byEmail: byEmail}, nil
}

func (f *playerFinder) findOrCreate(ctx context.Context, svc *Services, sp SeedPlayer) (uuid.UUID, error) {
	key := strings.ToLower(strings.TrimSpace(sp.Email))
	if id, ok := f.byEmail[key]; ok {
		return id, nil
	}
	email := sp.Email
	p, err := svc.Player.CreatePlayer(ctx, golf.CreatePlayerInput{
		FirstName: sp.FirstName, LastName: sp.LastName, Email: &email,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating player %s: %w", sp.Email, err)
	}
	f.byEmail[key] = p.ID
	return p.ID, nil
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
