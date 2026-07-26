package golf

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// --- fakes ---

type fakeMatchDB struct {
	match   *Match
	details []MatchDetail
}

func (f *fakeMatchDB) GetMatch(ctx context.Context, id uuid.UUID) (*Match, error) {
	return f.match, nil
}
func (f *fakeMatchDB) ListMatchesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Match, error) {
	return nil, nil
}
func (f *fakeMatchDB) ListMatchDetailsByTournament(ctx context.Context, tournamentID uuid.UUID) ([]MatchDetail, error) {
	return f.details, nil
}
func (f *fakeMatchDB) CreateMatch(ctx context.Context, in CreateMatchInput) (*Match, error) {
	return nil, nil
}

type fakeParticipantDB struct {
	participants []MatchParticipant
	withPlayers  []MatchParticipantPlayer
	deleteErr    error       // returned by DeleteMatchParticipant
	deleted      []uuid.UUID // player ids passed to DeleteMatchParticipant
}

func (f *fakeParticipantDB) ListMatchParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error) {
	return f.participants, nil
}
func (f *fakeParticipantDB) ListParticipantsWithPlayersByTournament(ctx context.Context, tournamentID uuid.UUID) ([]MatchParticipantPlayer, error) {
	return f.withPlayers, nil
}
func (f *fakeParticipantDB) CreateMatchParticipant(ctx context.Context, tournamentID, matchID, playerID, teamID uuid.UUID) (*MatchParticipant, error) {
	return nil, nil
}
func (f *fakeParticipantDB) DeleteMatchParticipant(ctx context.Context, matchID, playerID uuid.UUID) error {
	f.deleted = append(f.deleted, playerID)
	return f.deleteErr
}

type fakeScoreDB struct {
	scores        []Score
	saved         []Score
	recomputedFor []uuid.UUID
	recomputed    []StoredResult
}

func (f *fakeScoreDB) ListScoresByMatch(ctx context.Context, matchID uuid.UUID) ([]Score, error) {
	return f.scores, nil
}
func (f *fakeScoreDB) ListScoresByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Score, error) {
	return f.scores, nil
}

// SaveScoreAndRecompute mirrors the repository: the write is visible to the recompute,
// because both happen in one transaction.
func (f *fakeScoreDB) SaveScoreAndRecompute(ctx context.Context, s Score, tournamentID uuid.UUID, recompute func([]Score) StoredResult) error {
	f.saved = append(f.saved, s)
	f.scores = append(f.scores, s)
	f.recomputedFor = append(f.recomputedFor, s.MatchID)
	f.recomputed = append(f.recomputed, recompute(f.scores))
	return nil
}

type fakeResultDB struct{}

func (f *fakeResultDB) GetMatchResult(ctx context.Context, matchID uuid.UUID) (*StoredResult, error) {
	return nil, nil
}
func (f *fakeResultDB) ListMatchOutcomes(ctx context.Context, tournamentID uuid.UUID) ([]MatchOutcome, error) {
	return nil, nil
}
func (f *fakeResultDB) ListTournamentPlayerRecords(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]PlayerRecord, error) {
	return nil, nil
}
func (f *fakeResultDB) ListAllTournamentStandings(ctx context.Context) (map[uuid.UUID]TournamentStandings, error) {
	return nil, nil
}
func (f *fakeResultDB) ListCupData(ctx context.Context) (CupData, error) {
	return CupData{}, nil
}

func twoTeamMatch() (*fakeMatchDB, *fakeParticipantDB) {
	m := &fakeMatchDB{match: &Match{ID: matchID, TournamentID: tournamentID, CourseID: courseID, TeeColorID: teeColorID}}
	p := &fakeParticipantDB{participants: []MatchParticipant{
		{MatchID: matchID, TeamID: teamA, PlayerID: playerA},
		{MatchID: matchID, TeamID: teamB, PlayerID: playerB},
	}}
	return m, p
}

// --- tests ---

func TestSubmitScore_WritesScoreWithMatchCourseAndRecomputes(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	rdb := &fakeResultDB{}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: rdb}

	err := svc.SubmitScore(context.Background(), matchID, ScoreEntry{
		HoleNumber: 1, Strokes: 4, TeamID: teamA, PlayerID: pUUID(playerA),
	})
	if err != nil {
		t.Fatalf("SubmitScore: %v", err)
	}

	if len(sdb.saved) != 1 {
		t.Fatalf("want 1 score saved, got %d", len(sdb.saved))
	}
	got := sdb.saved[0]
	// course_id and tee_color_id are derived from the match, not the client.
	if got.CourseID != courseID || got.TeeColorID != teeColorID || got.MatchID != matchID {
		t.Errorf("score not stamped from match: %+v", got)
	}
	if got.TeamID != teamA || got.PlayerID == nil || *got.PlayerID != playerA || got.HoleNumber != 1 || got.Strokes != 4 {
		t.Errorf("score fields wrong: %+v", got)
	}
	// Recompute must happen in the same call — that is what makes the pair atomic.
	if len(sdb.recomputedFor) != 1 || sdb.recomputedFor[0] != matchID {
		t.Errorf("want recompute for the match, got %v", sdb.recomputedFor)
	}
}

func TestSubmitScore_RejectsTeamNotInMatch(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	rdb := &fakeResultDB{}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: rdb}

	err := svc.SubmitScore(context.Background(), matchID, ScoreEntry{
		HoleNumber: 1, Strokes: 4, TeamID: uuid.New(),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for team not in match, got %v", err)
	}
	if len(sdb.saved) != 0 || len(sdb.recomputedFor) != 0 {
		t.Error("must not write or recompute on validation failure")
	}
}

func TestListResults_AssemblesSidesProgressAndOutcome(t *testing.T) {
	m := &fakeMatchDB{details: []MatchDetail{{
		Match:      Match{ID: matchID, TournamentID: tournamentID, CourseID: courseID, TeeColorID: teeColorID},
		FormatName: "Singles",
		CourseName: "Test GC",
	}}}
	p := &fakeParticipantDB{withPlayers: []MatchParticipantPlayer{
		{MatchID: matchID, TeamID: teamA, PlayerID: playerA, FirstName: "Red", LastName: "One"},
		{MatchID: matchID, TeamID: teamB, PlayerID: playerB, FirstName: "Blue", LastName: "Two"},
	}}
	// Red wins holes 1-2 (4 vs 5); the match is still open (2 up, 16 to play).
	sdb := &fakeScoreDB{scores: []Score{
		{MatchID: matchID, TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{MatchID: matchID, TeamID: teamB, HoleNumber: 1, Strokes: 5},
		{MatchID: matchID, TeamID: teamA, HoleNumber: 2, Strokes: 4},
		{MatchID: matchID, TeamID: teamB, HoleNumber: 2, Strokes: 5},
	}}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: &fakeResultDB{}}

	results, err := svc.ListResults(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.FormatName != "Singles" || r.CourseName != "Test GC" {
		t.Errorf("display names wrong: %+v", r)
	}
	if r.Finished || r.WinnerTeamID != nil || r.Lead != 2 || r.HolesRemaining != 16 {
		t.Errorf("want open, 2 up with 16 to play: %+v", r)
	}
	if len(r.Sides) != 2 {
		t.Fatalf("want two sides, got %d", len(r.Sides))
	}
	if len(r.HoleResults) != 2 || r.HoleResults[0] == nil || *r.HoleResults[0] != teamA {
		t.Errorf("want two Red-won holes, got %+v", r.HoleResults)
	}
}

func TestHoleWinner(t *testing.T) {
	tests := []struct {
		name string
		a, b int32
		want *uuid.UUID
	}{
		{"team A lower", 4, 5, pUUID(teamA)},
		{"team B lower", 6, 5, pUUID(teamB)},
		{"halved", 4, 4, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := HoleWinner(HoleResult{TeamScores: []TeamHoleScore{
				{TeamID: teamA, Strokes: tc.a}, {TeamID: teamB, Strokes: tc.b},
			}})
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("want halved, got %v", *got)
			case tc.want != nil && (got == nil || *got != *tc.want):
				t.Fatalf("want %v, got %v", *tc.want, got)
			}
		})
	}
}

func TestRemoveParticipant_DelegatesToDB(t *testing.T) {
	p := &fakeParticipantDB{}
	svc := &MatchService{ParticipantDB: p}

	if err := svc.RemoveParticipant(context.Background(), matchID, playerA); err != nil {
		t.Fatalf("RemoveParticipant: %v", err)
	}
	if len(p.deleted) != 1 || p.deleted[0] != playerA {
		t.Fatalf("want delete for player %v, got %v", playerA, p.deleted)
	}
}

func TestRemoveParticipant_PropagatesNotFound(t *testing.T) {
	p := &fakeParticipantDB{deleteErr: ErrNotFound}
	svc := &MatchService{ParticipantDB: p}

	err := svc.RemoveParticipant(context.Background(), matchID, playerA)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// The results view drives a live leaderboard, which needs to know who is ahead in a match
// that is still being played — not just who won a finished one. Without it every client
// has to re-derive the leader by counting hole_results.
func TestListResults_ReportsTheLeaderOfAnUnfinishedMatch(t *testing.T) {
	m := &fakeMatchDB{details: []MatchDetail{{
		Match: Match{ID: matchID, TournamentID: tournamentID, CourseID: courseID, TeeColorID: teeColorID},
	}}}
	p := &fakeParticipantDB{withPlayers: []MatchParticipantPlayer{
		{MatchID: matchID, TeamID: teamA, PlayerID: playerA},
		{MatchID: matchID, TeamID: teamB, PlayerID: playerB},
	}}
	// Team A wins hole 1; the match is live, so there is a leader but no winner.
	sdb := &fakeScoreDB{scores: []Score{
		{MatchID: matchID, TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{MatchID: matchID, TeamID: teamB, HoleNumber: 1, Strokes: 5},
	}}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: &fakeResultDB{}}

	results, err := svc.ListResults(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}

	r := results[0]
	if r.LeaderTeamID == nil || *r.LeaderTeamID != teamA {
		t.Errorf("leader = %v, want team A", r.LeaderTeamID)
	}
	if r.WinnerTeamID != nil {
		t.Errorf("winner = %v, want none while unfinished", r.WinnerTeamID)
	}
}

// A halved hole leaves nobody ahead.
func TestListResults_NoLeaderWhenAllSquare(t *testing.T) {
	m := &fakeMatchDB{details: []MatchDetail{{
		Match: Match{ID: matchID, TournamentID: tournamentID, CourseID: courseID, TeeColorID: teeColorID},
	}}}
	p := &fakeParticipantDB{withPlayers: []MatchParticipantPlayer{
		{MatchID: matchID, TeamID: teamA, PlayerID: playerA},
		{MatchID: matchID, TeamID: teamB, PlayerID: playerB},
	}}
	sdb := &fakeScoreDB{scores: []Score{
		{MatchID: matchID, TeamID: teamA, HoleNumber: 1, Strokes: 4},
		{MatchID: matchID, TeamID: teamB, HoleNumber: 1, Strokes: 4},
	}}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: &fakeResultDB{}}

	results, err := svc.ListResults(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if results[0].LeaderTeamID != nil {
		t.Errorf("leader = %v, want none when all square", results[0].LeaderTeamID)
	}
}
