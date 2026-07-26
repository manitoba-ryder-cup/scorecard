package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

// TestShutdownErrIgnoresCleanTermination pins which errors count as a failed run.
// ListenAndServe always returns ErrServerClosed once Shutdown is called, so surfacing it
// would report every ordinary Cloud Run scale-down or redeploy as a crash.
func TestShutdownErrIgnoresCleanTermination(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"no error", nil, nil},
		{"server closed by Shutdown", http.ErrServerClosed, nil},
		{"signal context cancelled", context.Canceled, nil},
		{"wrapped server closed", fmt.Errorf("serving: %w", http.ErrServerClosed), nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shutdownErr(tc.err); !errors.Is(got, tc.want) {
				t.Errorf("shutdownErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestShutdownErrPropagatesRealFailures keeps the filter above narrow.
func TestShutdownErrPropagatesRealFailures(t *testing.T) {
	bindErr := errors.New("listen tcp :5000: bind: address already in use")

	if got := shutdownErr(bindErr); !errors.Is(got, bindErr) {
		t.Errorf("shutdownErr(%v) = %v, want it propagated", bindErr, got)
	}
}
