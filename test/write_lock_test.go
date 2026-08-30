package test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/manitoba-ryder-cup/scorecard/sdk"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
)

// scoreLandingNow opens a transaction that has done what a score submission does — taken the
// match's lock and written a score — and not yet committed. Racing two API calls and hoping
// for the interleaving does not work: whichever request starts first usually finishes first,
// and the window never opens. Holding it open from here makes it deterministic.
//
// The returned commit closes the transaction; the caller must call it.
func scoreLandingNow(t *testing.T, fix *util.Fixture) (commit func()) {
	t.Helper()
	ctx := context.Background()

	conn := util.ConnectAs(t, fix.TenantID)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// The same order the score write path takes: lock the match, then write against it.
	var locked string
	if err := tx.QueryRow(ctx, `SELECT id FROM matches WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
		fix.MatchID, fix.TenantID).Scan(&locked); err != nil {
		t.Fatalf("lock match: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO scores (match_id, team_id, player_id, course_id, tee_color_id, hole_number, tenant_id, strokes)
		 VALUES ($1, $2, $3, $4, $5, 1, $6, 4)`,
		fix.MatchID, fix.TeamRed, fix.RedPlayer, fix.CourseID, fix.TeeColorID, fix.TenantID); err != nil {
		t.Fatalf("write a score: %v", err)
	}

	return func() {
		if err := tx.Commit(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("commit the score: %v", err)
		}
	}
}

// refusedAfterTheScoreCommits runs write while a score is mid-flight and reports what the
// server answered once that score lands. A write that takes the match lock blocks until the
// commit and then sees the score; one that does not either answers before the score exists or
// decides on a read taken before it did.
func refusedAfterTheScoreCommits(t *testing.T, fix *util.Fixture, write func() error) error {
	t.Helper()
	commit := scoreLandingNow(t, fix)

	done := make(chan error, 1)
	go func() { done <- write() }()

	// Long enough for the write to have reached the point where it either blocks on the lock
	// or reads without one. Returning inside this window means it never waited.
	time.Sleep(500 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("the write answered (%v) while a score was uncommitted; it never took the lock", err)
	default:
	}

	commit()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("the write never returned after the score committed")
		return nil
	}
}

// Each of these reads whether the match has been scored and then writes on the answer. The
// lock is what stops the answer going stale in between, and these are what fail if it goes.
func TestAWriteCannotDecideWhileAScoreIsLanding(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("deleting the match", func(t *testing.T) {
		t.Parallel()
		client, fix := authedClient(t)
		err := refusedAfterTheScoreCommits(t, fix, func() error {
			return client.DeleteMatch(ctx, fix.MatchID)
		})
		wantsStatus(t, err, http.StatusConflict)
	})

	t.Run("moving the tee set", func(t *testing.T) {
		t.Parallel()
		client, fix := authedClient(t)
		gold := secondTeeSet(t, client, fix)
		err := refusedAfterTheScoreCommits(t, fix, func() error {
			_, err := client.UpdateMatch(ctx, fix.MatchID, sdk.UpdateMatchRequest{TeeColorID: &gold})
			return err
		})
		wantsStatus(t, err, http.StatusConflict)
	})

	t.Run("setting the lineup", func(t *testing.T) {
		t.Parallel()
		client, fix := authedClient(t)
		err := refusedAfterTheScoreCommits(t, fix, func() error {
			return setTheSameLineup(t, client, fix)
		})
		wantsStatus(t, err, http.StatusConflict)
	})
}
