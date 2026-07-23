package app

import (
	"github.com/manitoba-ryder-cup/scorecard/internal/db/postgres"
	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

// Services bundles the domain services wired over one database. The HTTP server and the
// CLI commands share it, so the dependency graph is assembled in exactly one place.
type Services struct {
	Player     *golf.PlayerService
	Match      *golf.MatchService
	Tournament *golf.TournamentService
	Course     *golf.CourseService
	Format     *golf.FormatService
	Roster     *golf.RosterService
	Team       *golf.TeamService
}

// NewServices constructs the repository adapters and wires the domain services.
func NewServices(db *postgres.DB) *Services {
	playersDB := postgres.NewPlayersDB(db)
	matchesDB := postgres.NewMatchesDB(db)
	participantsDB := postgres.NewParticipantsDB(db)
	scoresDB := postgres.NewScoresDB(db)
	teamsDB := postgres.NewTeamsDB(db)
	teamMembersDB := postgres.NewTeamMembersDB(db)
	tournamentsDB := postgres.NewTournamentsDB(db)
	resultsDB := postgres.NewResultsDB(db)
	teeColorsDB := postgres.NewTeeColorsDB(db)
	coursesDB := postgres.NewCoursesDB(db)
	teeSetsDB := postgres.NewTeeSetsDB(db)
	matchFormatsDB := postgres.NewMatchFormatsDB(db)
	tournamentPlayersDB := postgres.NewTournamentPlayersDB(db)

	teamService := &golf.TeamService{TeamDB: teamsDB}

	return &Services{
		Player: &golf.PlayerService{
			PlayerDB: playersDB,
			ResultDB: resultsDB,
		},
		Match: &golf.MatchService{
			MatchDB:       matchesDB,
			ParticipantDB: participantsDB,
			ScoreDB:       scoresDB,
			ResultDB:      resultsDB,
			HoleDB:        teeSetsDB,
		},
		Tournament: &golf.TournamentService{
			TournamentDB: tournamentsDB,
			ResultDB:     resultsDB,
			TeamService:  teamService,
		},
		Course: &golf.CourseService{
			TeeColorDB: teeColorsDB,
			CourseDB:   coursesDB,
			TeeSetDB:   teeSetsDB,
		},
		Format: &golf.FormatService{
			FormatDB: matchFormatsDB,
		},
		Roster: &golf.RosterService{
			TournamentPlayerDB: tournamentPlayersDB,
			TeamDB:             teamsDB,
			TeamMemberDB:       teamMembersDB,
			ResultDB:           resultsDB,
		},
		Team: teamService,
	}
}
