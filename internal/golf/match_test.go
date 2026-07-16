package golf

import (
	"context"
	"testing"
)

// --- fakes ---

type fakeMatchDB struct {
	match *Match
}

func (f *fakeMatchDB) GetMatch(ctx context.Context, id int32) (*Match, error) {
	return f.match, nil
}
func (f *fakeMatchDB) ListMatchesByTournament(ctx context.Context, tournamentID int32) ([]Match, error) {
	return nil, nil
}

type fakeParticipantDB struct {
	participants []MatchParticipant
}

func (f *fakeParticipantDB) ListMatchParticipants(ctx context.Context, matchID int32) ([]MatchParticipant, error) {
	return f.participants, nil
}

type fakeScoreDB struct {
	scores []Score
	saved  []Score
}

func (f *fakeScoreDB) ListScoresByMatch(ctx context.Context, matchID int32) ([]Score, error) {
	return f.scores, nil
}
func (f *fakeScoreDB) SaveScore(ctx context.Context, s Score) error {
	f.saved = append(f.saved, s)
	f.scores = append(f.scores, s) // make the write visible to the recompute read
	return nil
}

type fakeResultDB struct {
	upserted    []StoredResult
	upsertMatch []int32
}

func (f *fakeResultDB) UpsertMatchResult(ctx context.Context, matchID, tournamentID int32, r StoredResult) error {
	f.upsertMatch = append(f.upsertMatch, matchID)
	f.upserted = append(f.upserted, r)
	return nil
}
func (f *fakeResultDB) GetMatchResult(ctx context.Context, matchID int32) (*StoredResult, error) {
	return nil, nil
}
func (f *fakeResultDB) ListTeamPoints(ctx context.Context, tournamentID int32) (map[int32]float64, error) {
	return nil, nil
}
func (f *fakeResultDB) IsTournamentFinished(ctx context.Context, tournamentID int32) (bool, error) {
	return false, nil
}
func (f *fakeResultDB) GetPlayerRecord(ctx context.Context, playerID int32) (PlayerRecord, error) {
	return PlayerRecord{}, nil
}

func twoTeamMatch() (*fakeMatchDB, *fakeParticipantDB) {
	m := &fakeMatchDB{match: &Match{ID: 7, TournamentID: 3, CourseID: 11, TeeColorID: 2}}
	p := &fakeParticipantDB{participants: []MatchParticipant{
		{MatchID: 7, TeamID: 1, PlayerID: 100},
		{MatchID: 7, TeamID: 2, PlayerID: 200},
	}}
	return m, p
}

// --- tests ---

func TestSubmitScore_WritesScoreWithMatchCourseAndRecomputes(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	rdb := &fakeResultDB{}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: rdb}

	playerID := int32(100)
	err := svc.SubmitScore(context.Background(), 7, ScoreEntry{
		HoleNumber: 1, Strokes: 4, TeamID: 1, PlayerID: &playerID,
	})
	if err != nil {
		t.Fatalf("SubmitScore: %v", err)
	}

	if len(sdb.saved) != 1 {
		t.Fatalf("want 1 score saved, got %d", len(sdb.saved))
	}
	got := sdb.saved[0]
	// course_id and tee_color_id are derived from the match, not the client.
	if got.CourseID != 11 || got.TeeColorID != 2 || got.MatchID != 7 {
		t.Errorf("score not stamped from match: %+v", got)
	}
	if got.TeamID != 1 || got.PlayerID == nil || *got.PlayerID != 100 || got.HoleNumber != 1 || got.Strokes != 4 {
		t.Errorf("score fields wrong: %+v", got)
	}
	// The write must trigger a materialization recompute for this match.
	if len(rdb.upsertMatch) != 1 || rdb.upsertMatch[0] != 7 {
		t.Errorf("want recompute for match 7, got %v", rdb.upsertMatch)
	}
}

func TestSubmitScore_RejectsTeamNotInMatch(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	rdb := &fakeResultDB{}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: rdb}

	err := svc.SubmitScore(context.Background(), 7, ScoreEntry{
		HoleNumber: 1, Strokes: 4, TeamID: 99,
	})
	if err == nil {
		t.Fatal("want error for team not in match")
	}
	if len(sdb.saved) != 0 || len(rdb.upsertMatch) != 0 {
		t.Error("must not write or recompute on validation failure")
	}
}

func TestSubmitScore_RejectsInvalidStrokes(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: &fakeResultDB{}}

	err := svc.SubmitScore(context.Background(), 7, ScoreEntry{HoleNumber: 1, Strokes: 0, TeamID: 1})
	if err == nil {
		t.Fatal("want error for non-positive strokes")
	}
	if len(sdb.saved) != 0 {
		t.Error("must not write on validation failure")
	}
}

func TestSubmitScore_RejectsInvalidHole(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	svc := &MatchService{MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: &fakeResultDB{}}

	err := svc.SubmitScore(context.Background(), 7, ScoreEntry{HoleNumber: 19, Strokes: 4, TeamID: 1})
	if err == nil {
		t.Fatal("want error for hole out of range")
	}
	if len(sdb.saved) != 0 {
		t.Error("must not write on validation failure")
	}
}
