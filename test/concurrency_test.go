package test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
)

// TestConcurrentScoreSubmissionsMaterializeEveryHole covers the live-scoring hot path:
// two people entering their side's scores at once. Each submission persists a score then
// recomputes match_results from all of them, so unless that pair is serialized per match
// a submission holding an early snapshot can land last and revert the result to a partial
// view — every score row present, but standings derived from it wrong. Red wins every
// hole, so a stale result shows up as a match that is not finished.
func TestConcurrentScoreSubmissionsMaterializeEveryHole(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Several matches: the losing interleaving is timing dependent, so one would be a
	// coin flip.
	const matches = 8

	for i := range matches {
		client, fix := authedClient(t)

		type submission struct {
			team    uuid.UUID
			player  uuid.UUID
			strokes int32
		}

		var wg sync.WaitGroup
		errs := make(chan error, 36)
		for hole := int32(1); hole <= 18; hole++ {
			for _, s := range []submission{
				{fix.TeamRed, fix.RedPlayer, 4}, // Red wins every hole
				{fix.TeamBlue, fix.BluePlayer, 5},
			} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					errs <- client.SubmitScore(ctx, fix.MatchID, sdk.ScoreSubmission{
						HoleNumber: hole, Strokes: s.strokes, TeamID: s.team, PlayerID: &s.player,
					})
				}()
			}
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
			t.Errorf("match %d: finished = false after all 18 holes were scored; match_results is stale", i)
			continue
		}

		winner, err := client.GetMatchWinner(ctx, fix.MatchID)
		if err != nil {
			t.Fatalf("match %d: get winner: %v", i, err)
		}
		if winner.WinnerTeamID == nil || *winner.WinnerTeamID != fix.TeamRed {
			t.Errorf("match %d: winner = %v, want Red (%s)", i, winner.WinnerTeamID, fix.TeamRed)
		}
	}
}
