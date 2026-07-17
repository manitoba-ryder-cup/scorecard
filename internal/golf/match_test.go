package golf

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// --- fakes ---

type fakeMatchDB struct {
	match *Match
}

func (f *fakeMatchDB) GetMatch(ctx context.Context, id uuid.UUID) (*Match, error) {
	return f.match, nil
}
func (f *fakeMatchDB) ListMatchesByTournament(ctx context.Context, tournamentID uuid.UUID) ([]Match, error) {
	return nil, nil
}

type fakeParticipantDB struct {
	participants []MatchParticipant
}

func (f *fakeParticipantDB) ListMatchParticipants(ctx context.Context, matchID uuid.UUID) ([]MatchParticipant, error) {
	return f.participants, nil
}

type fakeScoreDB struct {
	scores []Score
	saved  []Score
}

func (f *fakeScoreDB) ListScoresByMatch(ctx context.Context, matchID uuid.UUID) ([]Score, error) {
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
func (f *fakeResultDB) GetPlayerRecord(ctx context.Context, playerID uuid.UUID) (PlayerRecord, error) {
	return PlayerRecord{}, nil
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
