package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
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
	Format   string   `json:"format"`
	TeeTimes []string `json:"tee_times"` // RFC3339
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
func SeedTournament(ctx context.Context, svc *Services, in *SeedInput) (*SeedSummary, error) {
	courseID, err := lookupCourse(ctx, svc, in.Course)
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

	tournament, err := svc.Tournament.CreateTournament(ctx, golf.CreateTournamentInput{
		Name: in.Tournament.Name, StartDate: start, EndDate: end, Location: in.Tournament.Location,
	})
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
	enteredByEmail := make(map[string]uuid.UUID, len(in.Players))
	for _, sp := range in.Players {
		playerID, err := finder.findOrCreate(ctx, svc, sp)
		if err != nil {
			return nil, err
		}
		if _, err := svc.Roster.EnterPlayer(ctx, golf.EnterPlayerInput{
			TournamentID: tournament.ID, PlayerID: playerID,
			Tier: defaultSeedTier(sp.Tier), Biography: sp.Biography, Hdcp: sp.Hdcp,
		}); err != nil {
			return nil, fmt.Errorf("entering %s: %w", sp.Email, err)
		}
		enteredByEmail[strings.ToLower(sp.Email)] = playerID
		summary.PlayersEntered++
	}

	// Draft each captain onto their team and set them as captain. The rest of the field
	// is entered but undrafted — the draft happens live.
	for color, email := range in.Captains {
		teamID, ok := teamByColor[color]
		if !ok {
			return nil, fmt.Errorf("tournament has no %q team", color)
		}
		captainID, ok := enteredByEmail[strings.ToLower(email)]
		if !ok {
			return nil, fmt.Errorf("%s captain %q is not in the roster", color, email)
		}
		if _, err := svc.Roster.DraftPlayer(ctx, teamID, captainID); err != nil {
			return nil, fmt.Errorf("drafting %s captain: %w", color, err)
		}
		if _, err := svc.Team.SetCaptain(ctx, teamID, captainID); err != nil {
			return nil, fmt.Errorf("setting %s captain: %w", color, err)
		}
	}

	// Matches per format, in schedule order (no participants — assigned live).
	for _, mg := range in.Matches {
		formatID, ok := formatIDs[mg.Format]
		if !ok {
			return nil, fmt.Errorf("unknown match format %q", mg.Format)
		}
		for _, tt := range mg.TeeTimes {
			teeTime, err := time.Parse(time.RFC3339, tt)
			if err != nil {
				return nil, fmt.Errorf("invalid tee_time %q: %w", tt, err)
			}
			if _, err := svc.Match.CreateMatch(ctx, golf.CreateMatchInput{
				TournamentID: tournament.ID, CourseID: courseID, TeeColorID: teeColorID,
				MatchFormatID: formatID, TeeTime: &teeTime,
			}); err != nil {
				return nil, fmt.Errorf("creating %s match: %w", mg.Format, err)
			}
			summary.Matches++
		}
	}
	return summary, nil
}

// defaultSeedTier mirrors the schema default rather than storing an empty tier.
func defaultSeedTier(tier string) string {
	if tier == "" {
		return "white"
	}
	return tier
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
	if sp.Email == "" {
		return uuid.Nil, fmt.Errorf("player %s %s has no email", sp.FirstName, sp.LastName)
	}
	key := strings.ToLower(sp.Email)
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

func lookupCourse(ctx context.Context, svc *Services, name string) (uuid.UUID, error) {
	courses, err := svc.Course.ListCourses(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("listing courses: %w", err)
	}
	for _, c := range courses {
		if c.Name == name {
			return c.ID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("course %q not found (create it first)", name)
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
