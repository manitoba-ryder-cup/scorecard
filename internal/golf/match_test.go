package golf

import (
	"context"
	"errors"
	"testing"
	"time"

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

// SaveScoresAndRecompute mirrors the repository: the guard sees the scores as they were
// before the write, and the recompute sees them after, because it is all one transaction.
func (f *fakeScoreDB) SaveScoresAndRecompute(
	ctx context.Context,
	matchID uuid.UUID,
	scores []Score,
	tournamentID uuid.UUID,
	guard func([]Score) error,
	recompute func([]Score) StoredResult,
) (StoredResult, error) {
	if err := guard(f.scores); err != nil {
		return StoredResult{}, err
	}
	for _, s := range scores {
		f.saved = append(f.saved, s)
		f.scores = upsert(f.scores, s)
	}
	f.recomputedFor = append(f.recomputedFor, matchID)
	result := recompute(f.scores)
	f.recomputed = append(f.recomputed, result)
	return result, nil
}

// upsert replaces the row for the same hole/team/player, matching the repo's ON CONFLICT
// write — appending instead would leave a stale score for best-ball to pick up.
func upsert(scores []Score, s Score) []Score {
	samePlayer := func(a, b *uuid.UUID) bool {
		if a == nil || b == nil {
			return a == b
		}
		return *a == *b
	}
	for i, existing := range scores {
		if existing.HoleNumber == s.HoleNumber && existing.TeamID == s.TeamID && samePlayer(existing.PlayerID, s.PlayerID) {
			scores[i] = s
			return scores
		}
	}
	return append(scores, s)
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

// duringTheCup is 10am local on the cup's first day — scores are only recordable on one
// of the tournament's days, so every write test needs the clock inside that window.
var duringTheCup = time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)

func matchService(m *fakeMatchDB, p *fakeParticipantDB, sdb *fakeScoreDB) *MatchService {
	return &MatchService{
		MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: &fakeResultDB{},
		TournamentDB: &fakeTournamentDB{tournament: &Tournament{
			ID: tournamentID, Name: "Manitoba Ryder Cup", StartDate: cupStart, EndDate: cupEnd,
		}},
		Now: func() time.Time { return duringTheCup },
	}
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
	svc := matchService(m, p, sdb)

	_, err := svc.SubmitHoleScores(context.Background(), matchID, 1, []ScoreEntry{{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4}})
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
	svc := matchService(m, p, sdb)

	_, err := svc.SubmitHoleScores(context.Background(), matchID, 1, []ScoreEntry{{TeamID: uuid.New(), PlayerID: nil, Strokes: 4}})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for team not in match, got %v", err)
	}
	if len(sdb.saved) != 0 || len(sdb.recomputedFor) != 0 {
		t.Error("must not write or recompute on validation failure")
	}
}

func TestSubmitHoleScores_WritesNothingWhenOneEntryIsInvalid(t *testing.T) {
	// The point of taking a hole as a unit: a bad entry anywhere in it rejects the whole
	// hole, so a client cannot leave one side scored and the other not.
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	svc := matchService(m, p, sdb)

	_, err := svc.SubmitHoleScores(context.Background(), matchID, 1, []ScoreEntry{
		{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4},
		{TeamID: uuid.New(), PlayerID: pUUID(playerB), Strokes: 5}, // not in this match
	})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for a team not in the match, got %v", err)
	}
	if len(sdb.saved) != 0 {
		t.Errorf("the valid entry must not land on its own, got %d writes", len(sdb.saved))
	}
}

func TestSubmitHoleScores_WritesTheWholeHoleAndRecomputesOnce(t *testing.T) {
	// One recompute for the hole, not one per score — the result the caller gets back
	// reflects every score in the request.
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	svc := matchService(m, p, sdb)

	got, err := svc.SubmitHoleScores(context.Background(), matchID, 1, []ScoreEntry{
		{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4},
		{TeamID: teamB, PlayerID: pUUID(playerB), Strokes: 5},
	})
	if err != nil {
		t.Fatalf("SubmitHoleScores: %v", err)
	}

	if len(sdb.saved) != 2 {
		t.Fatalf("want both scores written, got %d", len(sdb.saved))
	}
	if len(sdb.recomputedFor) != 1 {
		t.Errorf("want one recompute for the hole, got %d", len(sdb.recomputedFor))
	}
	// Both sides scored hole 1, so it counts: teamA 1 up with 17 to play.
	if got.Lead != 1 || got.LeaderTeamID == nil || *got.LeaderTeamID != teamA || got.HolesRemaining != 17 {
		t.Errorf("want teamA 1 up with 17 to play, got %+v", got)
	}
}

func TestSubmitHoleScores_RejectsAWriteBeforeTheTournament(t *testing.T) {
	// Nobody should be able to score a cup that is months away; the write is refused
	// rather than left to be noticed later in the standings.
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	svc := matchService(m, p, sdb)
	svc.Now = func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }

	_, err := svc.SubmitHoleScores(context.Background(), matchID, 1, []ScoreEntry{
		{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4},
	})

	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict before the tournament, got %v", err)
	}
	if len(sdb.saved) != 0 {
		t.Error("must not write outside the tournament's days")
	}
}

func TestSubmitHoleScores_AllowsTheLastGroupPastMidnightUTC(t *testing.T) {
	// The final group tees off mid-afternoon and finishes after midnight UTC. Reading the
	// tournament's dates as UTC would refuse their closing holes.
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	svc := matchService(m, p, sdb)
	svc.Now = func() time.Time { return time.Date(2026, 9, 20, 1, 20, 0, 0, time.UTC) }

	if _, err := svc.SubmitHoleScores(context.Background(), matchID, 17, []ScoreEntry{
		{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4},
	}); err != nil {
		t.Fatalf("want the last group's scores accepted, got %v", err)
	}
}

// decidedMatch is a match teamA has already won 10 & 8: holes 1-10 scored, teamA taking
// every one, so the lead can no longer be caught.
func decidedMatch() []Score {
	var scores []Score
	for h := int32(1); h <= 10; h++ {
		scores = append(scores,
			Score{MatchID: matchID, TeamID: teamA, PlayerID: pUUID(playerA), HoleNumber: h, Strokes: 4},
			Score{MatchID: matchID, TeamID: teamB, PlayerID: pUUID(playerB), HoleNumber: h, Strokes: 5},
		)
	}
	return scores
}

func TestSubmitScore_ReturnsTheRecomputedStatus(t *testing.T) {
	// The caller gets the new state back, so a client never has to re-derive the
	// close-out rule to know the score it just entered ended the match.
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{scores: decidedMatch()[:18]} // holes 1-9, teamA 9 up with 9 to play
	svc := matchService(m, p, sdb)

	got, err := svc.SubmitHoleScores(context.Background(), matchID, 10, []ScoreEntry{{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4}})
	if err != nil {
		t.Fatalf("SubmitScore: %v", err)
	}

	// teamB has not scored hole 10, so it isn't counted yet: still 9 up with 9 to play.
	if got.Finished || got.Lead != 9 || got.LeaderTeamID == nil || *got.LeaderTeamID != teamA {
		t.Errorf("want unfinished, teamA 9 up, got %+v", got)
	}
}

func TestSubmitScore_RejectsANewHoleOnAFinishedMatch(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{scores: decidedMatch()}
	svc := matchService(m, p, sdb)

	_, err := svc.SubmitHoleScores(context.Background(), matchID, 11, []ScoreEntry{{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4}})

	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict for a hole played after the close-out, got %v", err)
	}
	if len(sdb.saved) != 0 {
		t.Error("must not write past the hole the match was decided on")
	}
}

func TestSubmitScore_AllowsCorrectingAScoredHoleOnAFinishedMatch(t *testing.T) {
	// A typo can be what closed the match out early, so the holes that were played stay
	// correctable — otherwise the bad score is the reason it can't be fixed.
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{scores: decidedMatch()}
	svc := matchService(m, p, sdb)

	got, err := svc.SubmitHoleScores(context.Background(), matchID, 5, []ScoreEntry{{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 9}})
	if err != nil {
		t.Fatalf("want the correction accepted, got %v", err)
	}
	if len(sdb.saved) != 1 {
		t.Fatalf("want the correction written, got %d writes", len(sdb.saved))
	}
	// teamA now loses hole 5, so the lead drops and the match is no longer closed out.
	if got.Finished || got.Lead != 8 {
		t.Errorf("want the result recomputed to an open match 8 up, got %+v", got)
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
	svc := matchService(m, p, sdb)

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
	if r.Finished || r.Winner() != nil || r.Lead != 2 || r.HolesRemaining != 16 {
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
	svc := matchService(m, p, sdb)

	results, err := svc.ListResults(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}

	r := results[0]
	if r.LeaderTeamID == nil || *r.LeaderTeamID != teamA {
		t.Errorf("leader = %v, want team A", r.LeaderTeamID)
	}
	if r.Winner() != nil {
		t.Errorf("winner = %v, want none while unfinished", r.Winner())
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
	svc := matchService(m, p, sdb)

	results, err := svc.ListResults(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if results[0].LeaderTeamID != nil {
		t.Errorf("leader = %v, want none when all square", results[0].LeaderTeamID)
	}
}
