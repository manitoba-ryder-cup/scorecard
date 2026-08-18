package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manitoba-ryder-cup/scorecard/internal/golf"
)

func TestRespondDomainError_StatusMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not found", golf.ErrNotFound, http.StatusNotFound},
		{"invalid input", golf.ErrInvalidInput, http.StatusBadRequest},
		{"conflict", golf.ErrConflict, http.StatusConflict},
		// The sentinel is matched through wrapping, as repos/services wrap it.
		{"wrapped not found", fmt.Errorf("getting player 5: %w", golf.ErrNotFound), http.StatusNotFound},
		{"unknown error", errors.New("connection reset"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondDomainError(context.Background(), rec, "message", tc.err)
			if rec.Code != tc.want {
				t.Fatalf("want status %d, got %d", tc.want, rec.Code)
			}
		})
	}
}

func bodyOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body.Error
}

// A caller reads these. Reads are public, so the operation that failed would be told to
// any spectator, and it reads as a verdict on their request rather than a fault here.
func TestRespondError_CollapsesEveryServerFault(t *testing.T) {
	for _, message := range []string{"Failed to list players", "Failed to calculate match scores", "Failed to get teams data"} {
		rec := httptest.NewRecorder()
		respondError(context.Background(), rec, http.StatusInternalServerError, message, errors.New("connection reset"))
		if got := bodyOf(t, rec); got != serverFault {
			t.Errorf("%q reached the client as %q, want the one fault message", message, got)
		}
	}

	rec := httptest.NewRecorder()
	respondError(context.Background(), rec, http.StatusBadGateway, "Upstream is down", errors.New("boom"))
	if got := bodyOf(t, rec); got != serverFault {
		t.Errorf("502 reached the client as %q, want the one fault message", got)
	}
}

// A 4xx names something about the request, so it stays as the handler wrote it.
func TestRespondError_KeepsClientFacingMessages(t *testing.T) {
	for status, message := range map[int]string{
		http.StatusNotFound:   "Match not found",
		http.StatusBadRequest: "hole_number must be between 1 and 18",
		http.StatusConflict:   "Scoring is closed for this match",
	} {
		rec := httptest.NewRecorder()
		respondError(context.Background(), rec, status, message, errors.New("cause"))
		if got := bodyOf(t, rec); got != message {
			t.Errorf("status %d: got %q, want the message unchanged", status, got)
		}
	}
}
