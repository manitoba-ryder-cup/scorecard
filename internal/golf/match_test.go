package golf

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// --- fakes ---

type fakeMatchDB struct {
	match     *Match
	details   []MatchDetail
	updated   *UpdateMatchInput // what UpdateMatch was last handed
	deleted   *uuid.UUID        // what DeleteMatch was last handed
	deleteErr error
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
func (f *fakeMatchDB) UpdateMatch(ctx context.Context, in UpdateMatchInput) (*Match, error) {
	f.updated = &in
	return f.match, nil
}

func (f *fakeMatchDB) DeleteMatch(ctx context.Context, id uuid.UUID) error {
	f.deleted = &id
	return f.deleteErr
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
	resetFor      []uuid.UUID
}

func (f *fakeScoreDB) ResetMatch(ctx context.Context, matchID uuid.UUID) error {
	f.resetFor = append(f.resetFor, matchID)
	return nil
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

type fakeResultDB struct {
	outcomes    []MatchOutcome
	allOutcomes map[uuid.UUID][]MatchOutcome
}

func (f *fakeResultDB) GetMatchResult(ctx context.Context, matchID uuid.UUID) (*StoredResult, error) {
	return nil, nil
}
func (f *fakeResultDB) ListMatchOutcomes(ctx context.Context, tournamentID uuid.UUID) ([]MatchOutcome, error) {
	return f.outcomes, nil
}
func (f *fakeResultDB) ListAllMatchOutcomes(ctx context.Context) (map[uuid.UUID][]MatchOutcome, error) {
	return f.allOutcomes, nil
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

// teeOff is when the match under test goes out; duringTheRound is an hour later, inside
// the window every write test needs the clock to be in.
var (
	teeOff         = time.Date(2026, 9, 18, 13, 0, 0, 0, time.UTC)
	duringTheRound = teeOff.Add(time.Hour)
)

func matchService(m *fakeMatchDB, p *fakeParticipantDB, sdb *fakeScoreDB) *MatchService {
	return &MatchService{
		MatchDB: m, ParticipantDB: p, ScoreDB: sdb, ResultDB: &fakeResultDB{},
		Now: func() time.Time { return duringTheRound },
	}
}

func twoTeamMatch() (*fakeMatchDB, *fakeParticipantDB) {
	m := &fakeMatchDB{match: &Match{ID: matchID, TournamentID: tournamentID, CourseID: courseID, TeeColorID: teeColorID, TeeTime: teeOff}}
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
	// A bad entry rejects the whole hole, so one side cannot be left scored and the other not.
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
	// One recompute per hole, so the result returned reflects every score in the request.
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

func TestSubmitHoleScores_RejectsAWriteOutsideTheScoringWindow(t *testing.T) {
	// Measured from the tee time, so it is per-match and needs no timezone.
	for _, tc := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"months before the cup", teeOff.Add(-200 * 24 * time.Hour), false},
		{"the evening before", teeOff.Add(-15 * time.Hour), false},
		{"three hours out, before the window opens", teeOff.Add(-3 * time.Hour), false},
		{"exactly when it opens", teeOff.Add(-scoringOpensBefore), true},
		{"on the tee", teeOff, true},
		{"a slow round, six hours in", teeOff.Add(6 * time.Hour), true},
		{"corrections that evening", teeOff.Add(11 * time.Hour), true},
		{"exactly when it shuts", teeOff.Add(scoringClosesAfter), true},
		{"the next morning", teeOff.Add(20 * time.Hour), false},
		{"a year later", teeOff.Add(365 * 24 * time.Hour), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, p := twoTeamMatch()
			sdb := &fakeScoreDB{}
			svc := matchService(m, p, sdb)
			svc.Now = func() time.Time { return tc.now }

			_, err := svc.SubmitHoleScores(context.Background(), matchID, 1, []ScoreEntry{
				{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4},
			})

			if tc.want {
				if err != nil {
					t.Fatalf("want the score accepted, got %v", err)
				}
				return
			}
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("want ErrConflict, got %v", err)
			}
			if len(sdb.saved) != 0 {
				t.Error("must not write outside the scoring window")
			}
		})
	}
}

// The API publishes these two instants and the client gates its UI on them, so they have
// to be exactly the bounds the write path enforces. Divergence would be invisible from
// either side alone: the UI would offer a control the server refuses, or withhold one it
// would have accepted.
func TestScoringWindowIsTheWindowSubmitEnforces(t *testing.T) {
	opens, closes := ScoringWindow(teeOff)

	for _, tc := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"a second before the published open", opens.Add(-time.Second), false},
		{"exactly the published open", opens, true},
		{"midway between the published bounds", opens.Add(closes.Sub(opens) / 2), true},
		{"exactly the published close", closes, true},
		{"a second after the published close", closes.Add(time.Second), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := scoringOpen(tc.now, teeOff); got != tc.want {
				t.Errorf("scoringOpen(%s) = %v, want %v — published window is %s to %s",
					tc.now.Format(time.RFC3339), got, tc.want,
					opens.Format(time.RFC3339), closes.Format(time.RFC3339))
			}
		})
	}
}

// Both bounds hang off the tee time, so a match with no fixed start has no window either
// — worth pinning, because the client subtracts these from its own clock.
func TestScoringWindowStraddlesTheTeeTime(t *testing.T) {
	opens, closes := ScoringWindow(teeOff)
	if !opens.Before(teeOff) {
		t.Errorf("window should open before the tee time, got %s for a %s tee off", opens, teeOff)
	}
	if !closes.After(teeOff) {
		t.Errorf("window should close after the tee time, got %s for a %s tee off", closes, teeOff)
	}
}

func TestSubmitHoleScores_ScoresEachMatchOnItsOwnTeeTime(t *testing.T) {
	// A morning group's window shuts long before the afternoon group's opens.
	m, p := twoTeamMatch()
	m.match.TeeTime = teeOff.Add(20 * time.Hour) // out tomorrow morning
	sdb := &fakeScoreDB{}
	svc := matchService(m, p, sdb)

	_, err := svc.SubmitHoleScores(context.Background(), matchID, 1, []ScoreEntry{
		{TeamID: teamA, PlayerID: pUUID(playerA), Strokes: 4},
	})

	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict for a match not yet out, got %v", err)
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
	// The caller gets the new state back, so a client never re-derives the close-out rule.
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
	// A typo can be what closed the match out, so the holes it reached stay correctable.
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

	results, _, err := svc.ListResults(context.Background(), tournamentID)
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

func TestRemoveParticipant_PropagatesNotFound(t *testing.T) {
	p := &fakeParticipantDB{deleteErr: ErrNotFound}
	svc := &MatchService{ParticipantDB: p, ScoreDB: &fakeScoreDB{}}

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

	results, _, err := svc.ListResults(context.Background(), tournamentID)
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

// The phase the caller caches on, and where it comes from. Taken from the stored results
// rather than the scores this method already holds: /teams and the tournament record read
// those, and a match whose participant was removed keeps its result row while its scores
// cascade away — so deriving it here twice let one cup report two phases at once.
func TestListResults_TakesThePhaseFromTheStoredResults(t *testing.T) {
	tests := []struct {
		name     string
		outcomes []MatchOutcome
		want     sdk.TournamentPhase
	}{
		{"scheduled, nothing scored", []MatchOutcome{{}}, sdk.PhaseUpcoming},
		{"scored but undecided", []MatchOutcome{{Started: true}}, sdk.PhaseLive},
		{"closed out", []MatchOutcome{{Started: true, Finished: true}}, sdk.PhaseFinished},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := matchService(matchFixture())
			svc.ResultDB = &fakeResultDB{outcomes: tc.outcomes}

			_, phase, err := svc.ListResults(context.Background(), tournamentID)
			if err != nil {
				t.Fatalf("ListResults: %v", err)
			}
			if phase != tc.want {
				t.Errorf("phase = %v, want %v", phase, tc.want)
			}
		})
	}
}

// The disagreement itself: scores that read as a match still being played, against a
// stored result that closed out. The stored result decides, because it is what every
// other endpoint publishing the phase reads.
func TestListResults_PhaseFollowsTheStoredResultNotTheScores(t *testing.T) {
	m, p, sdb := matchFixture()
	sdb.scores = []Score{{MatchID: matchID, TeamID: teamA, HoleNumber: 1, Strokes: 4}}
	svc := matchService(m, p, sdb)
	svc.ResultDB = &fakeResultDB{outcomes: []MatchOutcome{{Started: true, Finished: true}}}

	_, phase, err := svc.ListResults(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if phase != sdk.PhaseFinished {
		t.Errorf("phase = %v, want finished", phase)
	}
}

// matchFixture is one singles match with both sides and no scores.
func matchFixture() (*fakeMatchDB, *fakeParticipantDB, *fakeScoreDB) {
	return &fakeMatchDB{details: []MatchDetail{{
			Match: Match{ID: matchID, TournamentID: tournamentID, CourseID: courseID, TeeColorID: teeColorID},
		}}},
		&fakeParticipantDB{withPlayers: []MatchParticipantPlayer{
			{MatchID: matchID, TeamID: teamA, PlayerID: playerA},
			{MatchID: matchID, TeamID: teamB, PlayerID: playerB},
		}},
		&fakeScoreDB{}
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

	results, _, err := svc.ListResults(context.Background(), tournamentID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if results[0].LeaderTeamID != nil {
		t.Errorf("leader = %v, want none when all square", results[0].LeaderTeamID)
	}
}

func TestResetMatch_WorksLongAfterTheWindowHasShut(t *testing.T) {
	m, p := twoTeamMatch()
	sdb := &fakeScoreDB{}
	svc := matchService(m, p, sdb)
	svc.Now = func() time.Time { return teeOff.Add(30 * 24 * time.Hour) }

	if err := svc.ResetMatch(context.Background(), matchID); err != nil {
		t.Fatalf("ResetMatch: %v", err)
	}
	if len(sdb.resetFor) != 1 || sdb.resetFor[0] != matchID {
		t.Errorf("reset = %v, want the match cleared once", sdb.resetFor)
	}
}

func TestRemoveParticipant_AllowedWhileTheMatchIsUnscored(t *testing.T) {
	m, p := twoTeamMatch()
	svc := matchService(m, p, &fakeScoreDB{})

	if err := svc.RemoveParticipant(context.Background(), matchID, playerA); err != nil {
		t.Fatalf("RemoveParticipant: %v", err)
	}
	if len(p.deleted) != 1 || p.deleted[0] != playerA {
		t.Errorf("removed = %v, want the player", p.deleted)
	}
}
