package app

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// recorder notes the order in which shutdown touches each dependency.
type recorder struct {
	calls *[]string
	err   error
}

func (r *recorder) ListenAndServe() error { return nil }

func (r *recorder) Shutdown(context.Context) error {
	*r.calls = append(*r.calls, "http")
	return r.err
}

func (r *recorder) Close() { *r.calls = append(*r.calls, "db") }

// TestShutdownDrainsHTTPBeforeClosingTheDatabase pins the ordering that makes shutdown
// graceful: Shutdown blocks until in-flight requests finish, and those requests are
// still querying, so closing the pool first turns a clean deploy into a burst of 500s.
func TestShutdownDrainsHTTPBeforeClosingTheDatabase(t *testing.T) {
	var calls []string
	server := &Server{
		httpServer: &recorder{calls: &calls},
		db:         &recorder{calls: &calls},
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	want := []string{"http", "db"}
	if !slices.Equal(calls, want) {
		t.Errorf("shutdown order = %v, want %v (the pool must outlive the requests draining against it)", calls, want)
	}
}

// TestShutdownClosesTheDatabaseEvenIfDrainingFails: a drain that times out must still
// release the pool.
func TestShutdownClosesTheDatabaseEvenIfDrainingFails(t *testing.T) {
	var calls []string
	drainErr := errors.New("drain timed out")
	server := &Server{
		httpServer: &recorder{calls: &calls, err: drainErr},
		db:         &recorder{calls: &calls},
	}

	err := server.Shutdown(context.Background())

	if !errors.Is(err, drainErr) {
		t.Errorf("shutdown err = %v, want %v", err, drainErr)
	}
	if !slices.Contains(calls, "db") {
		t.Errorf("shutdown order = %v, want the database closed despite the drain failing", calls)
	}
}
