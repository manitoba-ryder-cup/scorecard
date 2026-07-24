package golf

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// RosterService manages a tournament's roster: entering players with their
// per-tournament attributes (tier, biography, handicap), and the team draft
// (assigning entered players to teams).
type RosterService struct {
	TournamentPlayerDB tournamentPlayerDB
	TeamDB             teamDB
	TeamMemberDB       teamMemberDB
	ResultDB           resultDB
}

// EnterPlayerInput is the intent to enter a player in a tournament (or update their
// per-tournament attributes). Shape validation happens at the API boundary.
type EnterPlayerInput struct {
	TournamentID uuid.UUID
	PlayerID     uuid.UUID
	Tier         string
	Biography    string
	Hdcp         float32
}

// EnterPlayer enters a player in a tournament with their attributes. A duplicate
// entry surfaces as ErrConflict; an unknown player/tournament as ErrInvalidInput.
func (s *RosterService) EnterPlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error) {
	entry, err := s.TournamentPlayerDB.CreateTournamentPlayer(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to enter tournament player: %w", err)
	}
	return entry, nil
}

// UpdatePlayer updates an entered player's attributes. ErrNotFound if the player has
// not been entered in the tournament.
func (s *RosterService) UpdatePlayer(ctx context.Context, in EnterPlayerInput) (*TournamentPlayer, error) {
	entry, err := s.TournamentPlayerDB.UpdateTournamentPlayer(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to update tournament player: %w", err)
	}
	return entry, nil
}

// ListPlayers returns the tournament's entered players with their identity and their
// all-time record and cups (the roster view shows both).
func (s *RosterService) ListPlayers(ctx context.Context, tournamentID uuid.UUID) ([]TournamentPlayer, error) {
	entries, err := s.TournamentPlayerDB.ListTournamentPlayers(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tournament players: %w", err)
	}
	records, err := s.ResultDB.ListTournamentPlayerRecords(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list player records: %w", err)
	}
	cups, err := s.ResultDB.ListTournamentPlayerCups(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list player cups: %w", err)
	}
	for i := range entries {
		entries[i].Record = records[entries[i].PlayerID]
		entries[i].CupsWon = cups[entries[i].PlayerID]
	}
	return entries, nil
}

// DraftPlayer assigns an entered player to a team. The tournament is derived from the
// team (loaded first, so a bad team is a clean 404). Drafting a player who isn't
// entered surfaces as ErrInvalidInput; a player already drafted, as ErrConflict.
func (s *RosterService) DraftPlayer(ctx context.Context, teamID, playerID uuid.UUID) (*TeamMember, error) {
	team, err := s.TeamDB.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to load team: %w", err)
	}
	member, err := s.TeamMemberDB.CreateTeamMember(ctx, teamID, playerID, team.TournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to draft player: %w", err)
	}
	return member, nil
}

// UndraftPlayer removes a player from a team. ErrNotFound if they weren't on it. The
// team_members -> match_participants cascade also pulls them from any of that team's
// matches, so an undrafted player never lingers in a lineup.
func (s *RosterService) UndraftPlayer(ctx context.Context, teamID, playerID uuid.UUID) error {
	if err := s.TeamMemberDB.DeleteTeamMember(ctx, teamID, playerID); err != nil {
		return fmt.Errorf("failed to undraft player: %w", err)
	}
	return nil
}

// ListTeamMembers returns a team's drafted players with their identity — the same
// roster entry shape, filtered server-side to one team.
func (s *RosterService) ListTeamMembers(ctx context.Context, teamID uuid.UUID) ([]TournamentPlayer, error) {
	members, err := s.TournamentPlayerDB.ListTournamentPlayersByTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}
	return members, nil
}
