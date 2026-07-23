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

type fakeScoreDB struct {
	scores []Score
	saved  []Score
}

func (f *fakeScoreDB) ListScoresByMatch(ctx context.Context, matchID uuid.UUID) ([]Score, error) {
	return f.scores, nil
}
func (f *fakeScoreDB) ListScoresByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Score, error) {
	return f.scores, nil
}
func (f *fakeScoreDB) SaveScore(ctx context.Context, s Score) error {
	f.saved = append(f.saved, s)
	f.scores = append(f.scores, s) // make the write visible to the recompute read
	return nil
}

type fakeResultDB struct {
	upserted    []StoredResult
	upsertMatch []uuid.UUID
}

func (f *fakeResultDB) UpsertMatchResult(ctx context.Context, matchID, tournamentID uuid.UUID, r StoredResult) error {
	f.upsertMatch = append(f.upsertMatch, matchID)
	f.upserted = append(f.upserted, r)
	return nil
}
func (f *fakeResultDB) GetMatchResult(ctx context.Context, matchID uuid.UUID) (*StoredResult, error) {
	return nil, nil
}
func (f *fakeResultDB) ListTeamPoints(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]float64, error) {
	return nil, nil
}
func (f *fakeResultDB) IsTournamentFinished(ctx context.Context, tournamentID uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakeResultDB) ListTournamentPlayerRecords(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]PlayerRecord, error) {
	return nil, nil
}
func (f *fakeResultDB) ListTournamentPlayerCups(ctx context.Context, tournamentID uuid.UUID) (map[uuid.UUID]int, error) {
	return nil, nil
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
	// The write must trigger a materialization recompute for this match.
	if len(rdb.upsertMatch) != 1 || rdb.upsertMatch[0] != matchID {
		t.Errorf("want recompute for the match, got %v", rdb.upsertMatch)
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
	if len(sdb.saved) != 0 || len(rdb.upsertMatch) != 0 {
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
