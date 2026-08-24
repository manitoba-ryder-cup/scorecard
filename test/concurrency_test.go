package test

import (
	"context"
	"sync"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// TestConcurrentScoreSubmissionsMaterializeEveryHole covers the live-scoring hot path: holes
// landing at once from a retry, a second device, or a backlog going up when signal returns.
// Each submission persists its hole then recomputes match_results from every score, so unless
// that pair is serialized per match one holding an early snapshot can land last and revert the
// result to a partial view. Red wins every hole, so that shows up as a match not finished.
//
// Holes 1-10 only: that is the whole match at 10 & 8, and a finished match refuses scores for
// holes it never reached. Ten is also the earliest the lead can be decided, so no interleaving
// can finish the match while one of these holes is still unwritten.
//
//commentcap:allow -- the hole range is a constraint of the test, not an arbitrary choice
func TestConcurrentScoreSubmissionsMaterializeEveryHole(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Several matches: the losing interleaving is timing dependent, so one would be a
	// coin flip.
	const matches = 8

	for i := range matches {
		client, fix := authedClient(t)
		red, blue := fix.RedPlayer, fix.BluePlayer

		var wg sync.WaitGroup
		errs := make(chan error, 10)
		for hole := int32(1); hole <= 10; hole++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
					HoleNumber: hole,
					Scores: []sdk.ScoreEntry{
						{TeamID: fix.TeamRed, PlayerID: &red, Strokes: 4}, // Red wins every hole
						{TeamID: fix.TeamBlue, PlayerID: &blue, Strokes: 5},
					},
				})
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil {
				t.Fatalf("match %d: submit score: %v", i, err)
			}
		}

		// Every score is committed, so the result must reflect all of them.
		status, err := client.GetMatchStatus(ctx, fix.MatchID)
		if err != nil {
			t.Fatalf("match %d: get status: %v", i, err)
		}
		if !status.Finished {
			t.Errorf("match %d: finished = false after all 10 holes were scored; match_results is stale", i)
			continue
		}
		if status.WinnerTeamID == nil || *status.WinnerTeamID != fix.TeamRed {
			t.Errorf("match %d: winner = %v, want Red (%s)", i, status.WinnerTeamID, fix.TeamRed)
		}
	}
}
